package keymanager

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"custodial-wallet/pkg/crypto"
	"custodial-wallet/pkg/logger"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrInvalidKey       = errors.New("invalid key")
	ErrSignatureFailed  = errors.New("signature failed")
	ErrEncryptionFailed = errors.New("encryption failed")
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Service 密钥管理服务接口
type Service interface {
	GenerateMasterKey(userID uint, chain string) (*EncryptedKey, string, error)
	GenerateAddress(userID uint, chain string) (address string, derivationPath string, err error)
	GetKey(keyID uint) (*EncryptedKey, error)
	GetKeyByAddress(chain, address string) (*EncryptedKey, error)
	Sign(userID uint, chain, address string, txData []byte) ([]byte, error)
	SignWithRequestID(requestID string, userID uint, chain, address string, txData []byte) (*SignatureRequest, error)
	ListKeys(userID uint, chain string) ([]*EncryptedKey, error)
	ListSignatureRequests(userID uint, limit int) ([]*SignatureRequest, error)
}

type service struct {
	repo          Repository
	encryptionKey []byte
}

// NewService 创建密钥管理服务
func NewService(repo Repository, encryptionKey string) Service {
	// 从密码派生加密密钥
	key := crypto.SHA256([]byte(encryptionKey))
	keyBytes, _ := hex.DecodeString(key)
	return &service{
		repo:          repo,
		encryptionKey: keyBytes[:32],
	}
}

// GenerateMasterKey 生成主密钥
func (s *service) GenerateMasterKey(userID uint, chain string) (*EncryptedKey, string, error) {
	// 检查是否已有主密钥
	existing, err := s.repo.GetMasterKey(userID, chain)
	if err != nil {
		return nil, "", err
	}
	if existing != nil {
		return nil, "", errors.New("master key already exists")
	}

	// 生成助记词
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return nil, "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, "", err
	}

	// 生成种子
	seed := bip39.NewSeed(mnemonic, "")

	// 生成主密钥
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, "", err
	}

	// 加密私钥
	encryptedPriv, err := crypto.EncryptToBase64(masterKey.Key, s.encryptionKey)
	if err != nil {
		return nil, "", ErrEncryptionFailed
	}

	key := &EncryptedKey{
		UserID:        userID,
		Chain:         chain,
		PublicKey:     hex.EncodeToString(masterKey.PublicKey().Key),
		EncryptedPriv: encryptedPriv,
		KeyType:       "master",
		Status:        1,
	}

	if err := s.repo.CreateKey(key); err != nil {
		return nil, "", err
	}

	logger.Infof("Master key generated for user %d on chain %s", userID, chain)
	return key, mnemonic, nil
}

// GenerateAddress 生成地址
func (s *service) GenerateAddress(userID uint, chain string) (string, string, error) {
	// 获取主密钥
	masterKey, err := s.repo.GetMasterKey(userID, chain)
	if err != nil {
		return "", "", err
	}
	if masterKey == nil {
		// 自动创建主密钥
		masterKey, _, err = s.GenerateMasterKey(userID, chain)
		if err != nil {
			return "", "", err
		}
	}

	// 解密主密钥
	masterPrivBytes, err := crypto.DecryptFromBase64(masterKey.EncryptedPriv, s.encryptionKey)
	if err != nil {
		return "", "", ErrDecryptionFailed
	}

	// 获取派生索引
	index, err := s.repo.GetNextDerivationIndex(userID, chain)
	if err != nil {
		return "", "", err
	}

	// 派生路径: m/44'/coin'/0'/0/index
	coinType := s.getCoinType(chain)
	derivationPath := fmt.Sprintf("m/44'/%d'/0'/0/%d", coinType, index)

	// 派生子密钥
	master, err := bip32.NewMasterKey(masterPrivBytes)
	if err != nil {
		return "", "", err
	}

	// BIP44 派生
	purpose, _ := master.NewChildKey(bip32.FirstHardenedChild + 44)
	coinTypeKey, _ := purpose.NewChildKey(bip32.FirstHardenedChild + uint32(coinType))
	account, _ := coinTypeKey.NewChildKey(bip32.FirstHardenedChild + 0)
	change, _ := account.NewChildKey(0)
	addressKey, _ := change.NewChildKey(uint32(index))

	// 生成地址
	address, err := s.deriveAddress(chain, addressKey.Key)
	if err != nil {
		return "", "", err
	}

	// 加密派生私钥
	encryptedPriv, err := crypto.EncryptToBase64(addressKey.Key, s.encryptionKey)
	if err != nil {
		return "", "", ErrEncryptionFailed
	}

	// 保存派生密钥
	derivedKey := &EncryptedKey{
		UserID:         userID,
		Chain:          chain,
		PublicKey:      hex.EncodeToString(addressKey.PublicKey().Key),
		EncryptedPriv:  encryptedPriv,
		KeyType:        "derived",
		DerivationPath: derivationPath,
		Address:        address,
		Status:         1,
	}

	if err := s.repo.CreateKey(derivedKey); err != nil {
		return "", "", err
	}

	logger.Infof("Address generated: %s on %s for user %d", address, chain, userID)
	return address, derivationPath, nil
}

func (s *service) getCoinType(chain string) int {
	switch chain {
	case "bitcoin":
		return 0
	case "ethereum", "bsc", "polygon":
		return 60
	case "tron":
		return 195
	default:
		return 60
	}
}

func (s *service) deriveAddress(chain string, privateKey []byte) (string, error) {
	switch chain {
	case "ethereum", "bsc", "polygon":
		return s.deriveEthereumAddress(privateKey)
	case "tron":
		return s.deriveTronAddress(privateKey)
	case "bitcoin":
		return s.deriveBitcoinAddress(privateKey)
	default:
		return s.deriveEthereumAddress(privateKey)
	}
}

func (s *service) deriveEthereumAddress(privateKey []byte) (string, error) {
	privKey, err := ethcrypto.ToECDSA(privateKey)
	if err != nil {
		return "", err
	}
	address := ethcrypto.PubkeyToAddress(privKey.PublicKey)
	return address.Hex(), nil
}

func (s *service) deriveTronAddress(privateKey []byte) (string, error) {
	// Tron地址类似以太坊，但使用不同的前缀
	privKey, err := ethcrypto.ToECDSA(privateKey)
	if err != nil {
		return "", err
	}
	pubKey := privKey.Public().(*ecdsa.PublicKey)
	pubBytes := secp256k1.S256().Marshal(pubKey.X, pubKey.Y)
	hash := ethcrypto.Keccak256(pubBytes[1:])
	// Tron使用41前缀
	address := append([]byte{0x41}, hash[12:]...)
	return hex.EncodeToString(address), nil
}

func (s *service) deriveBitcoinAddress(privateKey []byte) (string, error) {
	// 简化的比特币地址生成
	privKey, err := ethcrypto.ToECDSA(privateKey)
	if err != nil {
		return "", err
	}
	pubKey := privKey.Public().(*ecdsa.PublicKey)
	pubBytes := secp256k1.S256().Marshal(pubKey.X, pubKey.Y)
	hash := crypto.SHA256(pubBytes)
	return hash[:40], nil
}

// GetKey 获取密钥
func (s *service) GetKey(keyID uint) (*EncryptedKey, error) {
	key, err := s.repo.GetKeyByID(keyID)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

// GetKeyByAddress 通过地址获取密钥
func (s *service) GetKeyByAddress(chain, address string) (*EncryptedKey, error) {
	key, err := s.repo.GetKeyByAddress(chain, address)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

// Sign 签名
func (s *service) Sign(userID uint, chain, address string, txData []byte) ([]byte, error) {
	key, err := s.repo.GetKeyByAddress(chain, address)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrKeyNotFound
	}
	if key.UserID != userID {
		return nil, errors.New("key does not belong to user")
	}

	// 解密私钥
	privateKey, err := crypto.DecryptFromBase64(key.EncryptedPriv, s.encryptionKey)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	// 签名
	privKey, err := ethcrypto.ToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	signature, err := ethcrypto.Sign(txData, privKey)
	if err != nil {
		return nil, ErrSignatureFailed
	}

	logger.Infof("Transaction signed for address %s on %s", address, chain)
	return signature, nil
}

// SignWithRequestID 带请求ID签名
func (s *service) SignWithRequestID(requestID string, userID uint, chain, address string, txData []byte) (*SignatureRequest, error) {
	key, err := s.repo.GetKeyByAddress(chain, address)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrKeyNotFound
	}

	// 创建签名请求
	req := &SignatureRequest{
		RequestID:   requestID,
		UserID:      userID,
		KeyID:       key.ID,
		Chain:       chain,
		RawTx:       hex.EncodeToString(txData),
		Status:      SignStatusPending,
		RequestedAt: time.Now(),
	}

	if err := s.repo.CreateSignatureRequest(req); err != nil {
		return nil, err
	}

	// 执行签名
	signature, err := s.Sign(userID, chain, address, txData)
	if err != nil {
		req.Status = SignStatusFailed
		req.ErrorMsg = err.Error()
		_ = s.repo.UpdateSignatureRequest(req)
		return req, err
	}

	now := time.Now()
	req.SignedTx = hex.EncodeToString(signature)
	req.Status = SignStatusSigned
	req.SignedAt = &now
	_ = s.repo.UpdateSignatureRequest(req)

	return req, nil
}

// ListKeys 列出密钥
func (s *service) ListKeys(userID uint, chain string) ([]*EncryptedKey, error) {
	return s.repo.ListKeysByUserID(userID, chain)
}

// ListSignatureRequests 列出签名请求
func (s *service) ListSignatureRequests(userID uint, limit int) ([]*SignatureRequest, error) {
	return s.repo.ListSignatureRequestsByUserID(userID, limit)
}

// GenerateRequestID 生成请求ID
func GenerateRequestID() string {
	return uuid.New().String()
}
