package preimage

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

// OracleClient implements the Oracle by writing the pre-image key to the given stream,
// and reading back a length-prefixed value.
type OracleClient struct {
	rw io.ReadWriter
}

func NewOracleClient(rw io.ReadWriter) *OracleClient {
	return &OracleClient{rw: rw}
}

var _ Oracle = (*OracleClient)(nil)

func (o *OracleClient) Get(key Key) []byte {
	h := key.PreimageKey()
	fmt.Print("HELLLLLLLLLLLLLLLLLLLLL")
	if _, err := o.rw.Write(h[:]); err != nil {
		panic(fmt.Errorf("failed to write key %s (%T) to pre-image oracle: %w", key, key, err))
	}

	var length uint64
	if err := binary.Read(o.rw, binary.BigEndian, &length); err != nil {
		panic(fmt.Errorf("failed to read pre-image length of key %s (%T) from pre-image oracle: %w", key, key, err))
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(o.rw, payload); err != nil {
		panic(fmt.Errorf("failed to read pre-image payload (length %d) of key %s (%T) from pre-image oracle: %w", length, key, key, err))
	}

	return payload
}

// OracleServer serves the pre-image requests of the OracleClient, implementing the same protocol as the onchain VM.
type OracleServer struct {
	rw io.ReadWriter
}

func NewOracleServer(rw io.ReadWriter) *OracleServer {
	return &OracleServer{rw: rw}
}

type PreimageGetter func(key [32]byte) ([]byte, error)

func (o *OracleServer) NextPreimageRequest(getPreimage PreimageGetter) error {
	var key [32]byte
	fmt.Println("GO: We read")
	if _, err := io.ReadFull(o.rw, key[:]); err != nil {
		fmt.Println("GO: We got an error: ", err)
		if err == io.EOF {
			return io.EOF
		}
		return fmt.Errorf("failed to read requested pre-image key: %w", err)
	}
	fmt.Println("GO: Getting preimage with hash: ", key)
	value, err := getPreimage(key)
	if err != nil {
		return fmt.Errorf("failed to serve pre-image %s request: %w", hex.EncodeToString(key[:]), err)
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(len(value)))
	fmt.Println("GO: We got length of ", uint64(len(value)))
	if _, err := o.rw.Write(b); err != nil {
		fmt.Println("GO: There has been an err, ", err)
		return fmt.Errorf("failed to write length-prefix %d: %w", len(value), err)
	}
	fmt.Println("GO: We finished writing, without error");
	fmt.Println("GO: we flush")
	if len(value) == 0 {
		fmt.Println("GO: Value is of length 0. We return Nil")
		return nil
	}
	fmt.Println("GO: We write the value, and check if there an error")
	if _, err := o.rw.Write(value); err != nil {
		fmt.Println("GO: we got an error: ", err)
		return fmt.Errorf("failed to write pre-image value (%d long): %w", len(value), err)
	}
	fmt.Println("GO: end of function, returning nil")
	return nil
}
