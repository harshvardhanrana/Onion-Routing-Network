package encryption

import (
	"fmt"
	"io"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rc4"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	ecies "github.com/ecies/go/v2"
)

const PAYLOADSIZE = 1024

type OnionCell struct {
	CellType   byte     // 1 byte (0 = Padding, 1 = Create, 2 = Data, 3 = Destroy)
	CircuitID  uint16   // 2 bytes (Circuit ID)
	Version    byte     // 1 byte
	BackF      byte     // 1 byte (0 = DES)
	ForwF      byte     // 1 byte (1 = RC4)
	Port       uint16   // 2 bytes
	IP         [4]byte  // 4 bytes (IPv4)
	Expiration uint32   // 4 bytes (UNIX timestamp)
	KeySeed    [16]byte // 16 bytes (128-bit key seed)
	Payload    []byte   // Variable length payload
}

func (cell OnionCell) String() string {
	return fmt.Sprintf(
		"CellType: %d\nCircuitID: %d\nVersion: %d\nBackF: %d\nForwF: %d\nPort: %d\nIP: %d.%d.%d.%d\nExpiration: %d\nKeySeed: %x\nPayload: %s",
		cell.CellType, cell.CircuitID, cell.Version, cell.BackF, cell.ForwF, cell.Port,
		cell.IP[0], cell.IP[1], cell.IP[2], cell.IP[3], cell.Expiration, cell.KeySeed, string(cell.Payload),
	)
}

func BuildMessage(cell OnionCell) []byte {
	data := make([]byte, 32+len(cell.Payload))

	data[0] = cell.CellType
	binary.BigEndian.PutUint16(data[1:3], cell.CircuitID)
	data[3] = cell.Version
	data[4] = cell.BackF
	data[5] = cell.ForwF
	binary.BigEndian.PutUint16(data[6:8], cell.Port)
	copy(data[8:12], cell.IP[:])
	binary.BigEndian.PutUint32(data[12:16], cell.Expiration)
	copy(data[16:32], cell.KeySeed[:])
	copy(data[32:], cell.Payload)

	return data
}

func RebuildMessage(data []byte) OnionCell {
	cell := OnionCell{}

	cell.CellType = data[0]
	cell.CircuitID = binary.BigEndian.Uint16(data[1:3])
	cell.Version = data[3]
	cell.BackF = data[4]
	cell.ForwF = data[5]
	cell.Port = binary.BigEndian.Uint16(data[6:8])
	copy(cell.IP[:], data[8:12])
	cell.Expiration = binary.BigEndian.Uint32(data[12:16])
	copy(cell.KeySeed[:], data[16:32])
	cell.Payload = data[32:]

	return cell
}

func DeriveKeys(seed []byte) ([]byte, []byte, []byte) {
	hash1 := sha256.Sum256(seed)     // First SHA-256 hash
	hash2 := sha256.Sum256(hash1[:]) // Second SHA-1 hash
	hash3 := sha256.Sum256(hash2[:]) // Third SHA-1 hash

	key1 := hash1[:8]  // First 8 bytes of first hash for DES
	key2 := hash2[:16] // First 16 bytes of second hash for RC4
	key3 := hash3[:16] // First 16 bytes of third hash for AES

	return key1, key2, key3
}

func EncryptRC4(data []byte, key []byte) []byte {
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		panic(err)
	}
	encrypted := make([]byte, len(data))
	cipher.XORKeyStream(encrypted, data)
	return encrypted
}

func EncryptAESCTR(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)

	encrypted := make([]byte, len(data))
	stream.XORKeyStream(encrypted, data)

	iv_appended_encryption := append(iv, encrypted...)

	return iv_appended_encryption, nil
}

func DecryptRC4(data []byte, key []byte) []byte {
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		panic(err)
	}
	decrypted := make([]byte, len(data))
	cipher.XORKeyStream(decrypted, data)
	return decrypted
}

func DecryptAESCTR(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	stream := cipher.NewCTR(block, iv)

	decrypted := make([]byte, len(ciphertext))
	stream.XORKeyStream(decrypted, ciphertext)

	return decrypted, nil
}

func EncryptRSA(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	encryptedData, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, nil)
	if err != nil {
		return nil, err
	}
	return encryptedData, nil
}

func DecryptRSA(encryptedData []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	decryptedData, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedData, nil)
	if err != nil {
		return nil, err
	}
	return decryptedData, nil
}

func EncryptECC(data []byte, publicKey *ecies.PublicKey) ([]byte, error) {
	return ecies.Encrypt(publicKey, data)
}

// DecryptECC decrypts data using the recipient's private ECC key
func DecryptECC(encryptedData []byte, privateKey *ecies.PrivateKey) ([]byte, error) {
	return ecies.Decrypt(privateKey, encryptedData)
}

func EncryptWithKey(ForwF byte, data []byte, key []byte) []byte {
	switch ForwF {
	case 1: // RC4
		return EncryptRC4(data, key)
	case 2: // AES
		encrypted, _ := EncryptAESCTR(data, key)
		return encrypted
	default:
		return nil
	}
}

func EncryptDataClient(data []byte, forward_keys [][]byte) []byte {
	for i := len(forward_keys) - 1; i >= 0; i-- {
		key := forward_keys[i]
		data = EncryptWithKey(1, data, key)
	}
	return data
}

func CreateCell(ip [4]byte, port uint16, payload []byte, circuitID uint16, keySeed [16]byte) OnionCell {

	cell := OnionCell{
		CellType:   1,          // Create cell
		CircuitID:  1001,       // Example Circuit ID
		Version:    1,          // Version 1
		BackF:      1,          // Backward cipher (e.g., DES)
		ForwF:      2,          // Forward cipher (e.g., RC4)
		Port:       port,       // Port number
		IP:         ip,         // Destination IP
		Expiration: 1700000000, // Expiration time
		// KeySeed:    [16]byte{'1', '6', 'B', 'y', 't', 'e', 's', 'K', 'e', 'y', 'S', 'e', 'e', 'd', '!'},
		KeySeed: keySeed, // Random key seed
		Payload: payload,            // Payload
	}
	return cell
}

func DataCell(payload []byte, circuitID uint16) OnionCell {
	key_seed := make([]byte, 16)
	rand.Read(key_seed)
	var ip [4]byte = [4]byte{0, 0, 0, 0}

	cell := OnionCell{
		CellType:   2,          // Create cell
		CircuitID:  circuitID,       // Example Circuit ID
		Version:    1,          // Version 1
		BackF:      1,          // Backward cipher (e.g., DES)
		ForwF:      2,          // Forward cipher (e.g., RC4)
		Port:       0,           // random
		IP:         ip,          // random
		Expiration: 1700000000, // Expiration time
		// KeySeed:    [16]byte{'1', '6', 'B', 'y', 't', 'e', 's', 'K', 'e', 'y', 'S', 'e', 'e', 'd', '!'},
		KeySeed: [16]byte(key_seed), // Random key seed
		Payload: payload,            // Payload
	}
	return cell
}




func main() {
	// cell := OnionCell{
	// 	CellType:   1,                    // Create cell
	// 	CircuitID:  1001,                 // Example Circuit ID
	// 	Version:    1,                    // Version 1
	// 	BackF:      1,                    // Backward cipher (e.g., DES)
	// 	ForwF:      2,                    // Forward cipher (e.g., RC4)
	// 	Port:       9002,                 // Port number
	// 	IP:         [4]byte{192, 168, 1, 1}, // Destination IP
	// 	Expiration: 1700000000,           // Expiration time
	// 	KeySeed:    [16]byte{'1', '6', 'B', 'y', 't', 'e', 's', 'K', 'e', 'y', 'S', 'e', 'e', 'd', '!'},
	// 	Payload:    []byte("Hello, Onion!"), // Payload
	// }

	key_seed := make([]byte, 16)
	rand.Read(key_seed)
	cell := CreateCell([4]byte{192, 168, 1, 1}, 9002, []byte("Hello, Onion!"), 1001, [16]byte(key_seed))

	message := BuildMessage(cell)

	fmt.Printf("Built Message:\n%x\n", message)

	fmt.Printf("Size of message: %d bytes\n", len(message))

	rebuiltCell := RebuildMessage(message)

	fmt.Printf("Rebuilt Cell:\n")
	fmt.Println(rebuiltCell.String())

	// check rc4
	_, key2, _ := DeriveKeys(cell.KeySeed[:])
	encryptedPayload := EncryptRC4(cell.Payload, key2)
	fmt.Printf("Encrypted Payload: %x\n", encryptedPayload)
	decryptedPayload := DecryptRC4(encryptedPayload, key2)
	fmt.Printf("Decrypted Payload: %s\n", decryptedPayload)

	// check aes
	encryptedPayloadAES, _ := EncryptAESCTR(cell.Payload, key2)
	fmt.Printf("Encrypted Payload AES: %x\n", encryptedPayloadAES)
	decryptedPayloadAES, _ := DecryptAESCTR(encryptedPayloadAES, key2)
	fmt.Printf("Decrypted Payload AES: %s\n", decryptedPayloadAES)

	// check rsa
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Error generating RSA key:", err)
		return
	}
	publicKey := &privateKey.PublicKey
	encryptedData, err := EncryptRSA(cell.Payload, publicKey)
	if err != nil {
		fmt.Println("Error encrypting data:", err)
		return
	}
	decryptedData, err := DecryptRSA(encryptedData, privateKey)
	if err != nil {
		fmt.Println("Error decrypting data:", err)
		return
	}
	fmt.Printf("Encrypted Data: %x\n", encryptedData)
	fmt.Printf("Decrypted Data: %s\n", decryptedData)

	// check ecc
	// priv, pub, err := genECCKeyPair()
	// encryptedECCData, err := EncryptECC(cell.Payload, pub)
	// if err != nil {
	// 	fmt.Println("Error encrypting ECC data:", err)
	// 	return
	// }
	// decryptedECCData, err := DecryptECC(encryptedECCData, priv)
	// if err != nil {
	// 	fmt.Println("Error decrypting ECC data:", err)
	// 	return
	// }
	// fmt.Printf("Encrypted ECC Data: %x\n", encryptedECCData)
	// fmt.Printf("Decrypted ECC Data: %s\n", decryptedECCData)

}
