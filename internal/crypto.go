package internal

import (
	"crypto/md5"
	"fmt"
	"hash/fnv"
)

// Hash is a simple wrapper around the MD5 algorithm implementation in the
// Go standard library. It takes in data and a salt and returns the hashed
// representation.
func Hash(data string, salt string) string {
	output := md5.Sum([]byte(data + salt))
	return fmt.Sprintf("%x", output)
}

// Fnv is a non-cryptographic hashing function based on non-cryptographic hash
// functions created by Glenn Fowler, Landon Curt Noll, and Phong Vo. This
// wraps the standard library FNV hash function by representing it as a 32 bit
// integer. Then it takes that number and "hashes" it by adding the rune's
// unicode value (as a uint 32) and then takes the modulus of 26, letting the number
// be represented as a letter of the alphabet. This is not a cryptographically
// secure operation, it is purely to replace numbers with a human-readable string
// to satisfy the requirement that any vhost with a "/" in it cannot end in a number
// (to avoid someone obtaining a vhost that is a cidr mask, it can cause issues).
func Fnv(data string) string {
	h := fnv.New32()
	h.Write([]byte(data))

	hash := h.Sum32()

	alphabet := "abcdefghijklmnopqrstuvwxyz"
	res := ""

	for _, char := range fmt.Sprintf("%d", hash) {
		res = res + string(alphabet[(uint32(char)+hash)%26])

		hash = (hash << 1) | (hash >> 31) // "rotate" the number for extra variance
	}

	return res
}
