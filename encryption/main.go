package main

import (
	"encoding/binary"
	"fmt"
	"crypto/sha256"
	// "net"
)

type OnionCell struct {
	CellType   byte   // 1 byte (0 = Padding, 1 = Create, 2 = Data, 3 = Destroy)
	CircuitID  uint16 // 2 bytes (Circuit ID)
	Version    byte   // 1 byte
	BackF      byte   // 1 byte (0 = DES)
	ForwF      byte   // 1 byte (1 = RC4)
	Port       uint16 // 2 bytes
	IP         [4]byte // 4 bytes (IPv4)
	Expiration uint32  // 4 bytes (UNIX timestamp)
	KeySeed    [16]byte // 16 bytes (128-bit key seed)
	Payload    []byte  // Variable length payload
}

func (cell OnionCell) String() string {
	return fmt.Sprintf(
		"CellType: %d\nCircuitID: %d\nVersion: %d\nBackF: %d\nForwF: %d\nPort: %d\nIP: %d.%d.%d.%d\nExpiration: %d\nKeySeed: %x\nPayload: %s",
		cell.CellType, cell.CircuitID, cell.Version, cell.BackF, cell.ForwF, cell.Port,
		cell.IP[0], cell.IP[1], cell.IP[2], cell.IP[3], cell.Expiration, cell.KeySeed, string(cell.Payload),
	)
}

func buildMessage(cell OnionCell) []byte {
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

func rebuildMessage(data []byte) OnionCell {
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

func deriveKeys(seed []byte) (key1, key2, key3 []byte) {
	hash1 := sha256.Sum256(seed) // First SHA-256 hash
	hash2 := sha256.Sum256(hash1[:]) // Second SHA-1 hash
	hash3 := sha256.Sum256(hash2[:]) // Third SHA-1 hash

	key1 = hash1[:8]   // First 8 bytes of first hash for DES
	key2 = hash2[:16]  // First 16 bytes of second hash for RC4
	key3 = hash3[:16]  // First 16 bytes of third hash for AES

	return key1, key2, key3
}


func main() {
	cell := OnionCell{
		CellType:   1,                    // Create cell
		CircuitID:  1001,                 // Example Circuit ID
		Version:    1,                    // Version 1
		BackF:      1,                    // Backward cipher (e.g., DES)
		ForwF:      2,                    // Forward cipher (e.g., RC4)
		Port:       9002,                 // Port number
		IP:         [4]byte{192, 168, 1, 1}, // Destination IP
		Expiration: 1700000000,           // Expiration time
		KeySeed:    [16]byte{'1', '6', 'B', 'y', 't', 'e', 's', 'K', 'e', 'y', 'S', 'e', 'e', 'd', '!'},
		Payload:    []byte("Hello, Onion!"), // Payload
	}

	message := buildMessage(cell)

	fmt.Printf("Built Message:\n%x\n", message)

	fmt.Printf("Size of message: %d bytes\n", len(message))

	rebuiltCell := rebuildMessage(message)

	fmt.Printf("Rebuilt Cell:\n")
	fmt.Println(rebuiltCell.String())

}
