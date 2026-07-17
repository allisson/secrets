package keyring

import "context"

// FakeKMSService is an in-memory KMSService for unit tests. It hands out a
// fakeKMSKeeper whose Encrypt/Decrypt are a reversible byte transform — no real
// KMS, no network. It is the second adapter that makes the KMSService/KMSKeeper
// interfaces a real seam: gocloud.dev in production, this in tests.
//
// The Fail* fields, when non-nil, force the matching operation to return the
// stored error, for exercising failure paths in callers.
type FakeKMSService struct {
	FailOpen    error
	FailEncrypt error
	FailDecrypt error
	FailClose   error
}

// NewFakeKMSService returns a ready-to-use FakeKMSService.
func NewFakeKMSService() *FakeKMSService {
	return &FakeKMSService{}
}

// OpenKeeper returns a fake keeper, or FailOpen if it is set.
func (f *FakeKMSService) OpenKeeper(_ context.Context, _ string) (KMSKeeper, error) {
	if f.FailOpen != nil {
		return nil, f.FailOpen
	}
	return &fakeKMSKeeper{
		failEncrypt: f.FailEncrypt,
		failDecrypt: f.FailDecrypt,
		failClose:   f.FailClose,
	}, nil
}

type fakeKMSKeeper struct {
	failEncrypt error
	failDecrypt error
	failClose   error
}

// fakeKMSMask is the byte the fake XORs data with. XOR is its own inverse, so
// Decrypt(Encrypt(x)) == x, giving a deterministic round-trip without real keys.
const fakeKMSMask = 0x5a

func (k *fakeKMSKeeper) Encrypt(_ context.Context, plaintext []byte) ([]byte, error) {
	if k.failEncrypt != nil {
		return nil, k.failEncrypt
	}
	return xorMask(plaintext), nil
}

func (k *fakeKMSKeeper) Decrypt(_ context.Context, ciphertext []byte) ([]byte, error) {
	if k.failDecrypt != nil {
		return nil, k.failDecrypt
	}
	return xorMask(ciphertext), nil
}

func (k *fakeKMSKeeper) Close() error {
	return k.failClose
}

func xorMask(in []byte) []byte {
	out := make([]byte, len(in))
	for i := range in {
		out[i] = in[i] ^ fakeKMSMask
	}
	return out
}
