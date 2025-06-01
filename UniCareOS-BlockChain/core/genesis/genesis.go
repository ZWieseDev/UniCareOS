package genesis

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "time"

    "unicareos/core/block"
    "unicareos/types/ids"
)


var GenesisMemoryDescription = "In the beginning, Aeon remembered the first light."

// LoadGenesisConfig loads the genesis config from genesis.json
func LoadGenesisConfig(path string) (*GenesisConfig, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("could not open genesis config: %w", err)
    }
    defer file.Close()
    bytes, err := ioutil.ReadAll(file)
    if err != nil {
        return nil, fmt.Errorf("could not read genesis config: %w", err)
    }
    var config GenesisConfig
    if err := json.Unmarshal(bytes, &config); err != nil {
        return nil, fmt.Errorf("could not parse genesis config: %w", err)
    }
    return &config, nil
}

// CreateGenesisBlockFromConfig builds the genesis block from a loaded config
func CreateGenesisBlockFromConfig(cfg *GenesisConfig) block.Block {
    const SignatureThreshold = 2 // Example threshold for testing
    fmt.Printf("[Genesis] Found %d signatures, threshold required: %d\n", len(cfg.Signatures), SignatureThreshold)
    if len(cfg.Signatures) < SignatureThreshold {
        fmt.Printf("[Genesis] ERROR: Not enough signatures to create genesis block!\n")
        AppendAuditEvent(AuditEvent{
            Timestamp: time.Now().UTC(),
            EventType: "signature_check",
            Details: mustMarshalJSON(map[string]interface{}{
                "signatures": cfg.Signatures,
                "threshold": SignatureThreshold,
                "result": "failed",
            }),
        })
        return block.Block{} // Return empty block to indicate failure
    }
    AppendAuditEvent(AuditEvent{
        Timestamp: time.Now().UTC(),
        EventType: "signature_check",
        Details: mustMarshalJSON(map[string]interface{}{
            "signatures": cfg.Signatures,
            "threshold": SignatureThreshold,
            "result": "passed",
        }),
    })
    AppendAuditEvent(AuditEvent{
        Timestamp: time.Now().UTC(),
        EventType: "validator_set",
        Details: mustMarshalJSON(map[string]interface{}{
            "validators": cfg.InitialValidators,
        }),
    })
    AppendAuditEvent(AuditEvent{
        Timestamp: time.Now().UTC(),
        EventType: "initial_params",
        Details: mustMarshalJSON(map[string]interface{}{
            "params": cfg.InitialParams,
        }),
    })
    // Use config values, fallback to dummy if needed
    genesisTime := cfg.GenesisTime
    description := GenesisMemoryDescription
    validatorDID := "did:unicare:dummy"
    if len(cfg.InitialValidators) > 0 {
        validatorDID = cfg.InitialValidators[0].DID
        // validatorPubKey is loaded but not used in block creation yet
    }
    fmt.Printf("[Genesis] Creating genesis block with ValidatorDID: %s at %s\n", validatorDID, genesisTime.Format(time.RFC3339Nano))

    eventIDSeed := []byte(description + genesisTime.Format(time.RFC3339Nano))
    genesisEvent := block.ChainedEvent{
        EventID:     ids.NewID(eventIDSeed),
        EventType:   "genesis",
        Description: description,
        Timestamp: genesisTime,
    }
    blk := block.Block{
        Version:         cfg.InitialParams.ProtocolVersion,
        ProtocolVersion: cfg.InitialParams.ProtocolVersion,
        Height:          0,
        PrevHash:        "",
        MerkleRoot:      cfg.InitialSchemaHash, // Placeholder for now
        Timestamp:       genesisTime,
        ValidatorDID:    validatorDID,
        OpUnitsUsed:     0,
        Events:          []block.ChainedEvent{genesisEvent},
        ExtraData:       nil,
        ParentGasUsed:   0,
        StateRoot:       "",
    }
    blk.BlockID = blk.ComputeID()
    AppendAuditEvent(AuditEvent{
        Timestamp: time.Now().UTC(),
        EventType: "block_created",
        Details: mustMarshalJSON(map[string]interface{}{
            "blockID": fmt.Sprintf("%x", blk.BlockID[:]),
            "validator": blk.ValidatorDID,
            "timestamp": blk.Timestamp,
        }),
    })
    // Compute and print Merkle root of the audit log
    // Compute and print Merkle root of the audit log, and anchor in block
    if root, err := ComputeAuditLogMerkleRoot(); err == nil {
        fmt.Printf("[Audit] Merkle root of audit log: %s\n", root)
        blk.ExtraData = []byte(root) // Anchor Merkle root in genesis block
        AppendAuditEvent(AuditEvent{
            Timestamp: time.Now().UTC(),
            EventType: "merkle_root_anchored",
            Details: mustMarshalJSON(map[string]interface{}{
                "merkleRoot": root,
                "anchored": true,
            }),
        })
    } else {
        fmt.Printf("[Audit] ERROR computing audit log Merkle root: %v\n", err)
        AppendAuditEvent(AuditEvent{
            Timestamp: time.Now().UTC(),
            EventType: "merkle_root_anchored",
            Details: mustMarshalJSON(map[string]interface{}{
                "error": err.Error(),
                "anchored": false,
            }),
        })
    }
    return blk
}

// Deprecated: Use CreateGenesisBlockFromConfig instead
func CreateGenesisBlock() block.Block {
    cfg, err := LoadGenesisConfig("genesis.json")
    if err != nil {
        AppendAuditEvent(AuditEvent{
            Timestamp: time.Now().UTC(),
            EventType: "config_load_error",
            Details: mustMarshalJSON(map[string]interface{}{
                "success": false,
                "error": fmt.Sprintf("%v", err),
            }),
        })
    } else {
        AppendAuditEvent(AuditEvent{
            Timestamp: time.Now().UTC(),
            EventType: "config_load",
            Details: mustMarshalJSON(map[string]interface{}{
                "success": true,
            }),
        })
    }
    if err != nil {
        fmt.Printf("[Genesis] WARNING: %v\nUsing legacy hardcoded values.\n", err)
        // fallback to legacy
        genesisTime := time.Date(2024, 4, 30, 0, 0, 0, 0, time.UTC)
        eventIDSeed := []byte(GenesisMemoryDescription + genesisTime.Format(time.RFC3339Nano))
        genesisEvent := block.ChainedEvent{
            EventID:     ids.NewID(eventIDSeed),
            EventType:   "genesis",
            Description: GenesisMemoryDescription,
            Timestamp: genesisTime,
        }
        block := block.Block{
            Version:         "1.0.0",
            ProtocolVersion: "1.0.0",
            Height:          0,
            PrevHash:        "",
            MerkleRoot:      "",
            Timestamp:       genesisTime,
            ValidatorDID:    "did:unicare:genesis",
            OpUnitsUsed:     0,
            Events:          []block.ChainedEvent{genesisEvent},
            ExtraData:       nil,
            ParentGasUsed:   0,
            StateRoot:       "",
        }
        block.BlockID = block.ComputeID()
        return block
    }
    return CreateGenesisBlockFromConfig(cfg)
}