package cannon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

const (
	proofsDir = "proofs"
)

type proofData struct {
	ClaimValue hexutil.Bytes `json:"post"`
	StateData  hexutil.Bytes `json:"state-data"`
	ProofData  hexutil.Bytes `json:"proof-data"`
}

type ProofGenerator interface {
	// GenerateProof executes cannon to generate a proof at the specified trace index in dataDir.
	GenerateProof(dataDir string, proofAt uint64) error
}

type CannonTraceProvider struct {
	dir       string
	generator ProofGenerator
}

func NewCannonTraceProvider(logger log.Logger, cfg *config.Config) *CannonTraceProvider {
	return &CannonTraceProvider{
		dir:       cfg.CannonDatadir,
		generator: NewExecutor(logger, cfg),
	}
}

func (p *CannonTraceProvider) Get(i uint64) (common.Hash, error) {
	proof, err := p.loadProof(i)
	if err != nil {
		return common.Hash{}, err
	}
	value := common.BytesToHash(proof.ClaimValue)

	if value == (common.Hash{}) {
		return common.Hash{}, errors.New("proof missing post hash")
	}
	return value, nil
}

func (p *CannonTraceProvider) GetPreimage(i uint64) ([]byte, []byte, error) {
	proof, err := p.loadProof(i)
	if err != nil {
		return nil, nil, err
	}
	value := ([]byte)(proof.StateData)
	if len(value) == 0 {
		return nil, nil, errors.New("proof missing state data")
	}
	data := ([]byte)(proof.ProofData)
	if len(data) == 0 {
		return nil, nil, errors.New("proof missing proof data")
	}
	return value, data, nil
}

func (p *CannonTraceProvider) AbsolutePreState() []byte {
	panic("absolute prestate not yet supported")
}

func (p *CannonTraceProvider) loadProof(i uint64) (*proofData, error) {
	path := filepath.Join(p.dir, proofsDir, fmt.Sprintf("%d.json", i))
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := p.generator.GenerateProof(p.dir, i); err != nil {
			return nil, fmt.Errorf("generate cannon trace with proof at %v: %w", i, err)
		}
		// Try opening the file again now and it should exist.
		file, err = os.Open(path)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot open proof file (%v): %w", path, err)
	}
	defer file.Close()
	var proof proofData
	err = json.NewDecoder(file).Decode(&proof)
	if err != nil {
		return nil, fmt.Errorf("failed to read proof (%v): %w", path, err)
	}
	return &proof, nil
}
