# UniCareOS: Build Blueprint & Guardrail System

## 1. Chain Metadata
- **Chain Name**: UniCareOS
- **Chain ID**: aeon_root_chain
- **Consensus Type**: Proof of Stake (Avalanche Subnet)
- **Primary Language**: Go 1.21.x

## 2. Core Modules
- `ValidatorPool`
  - var: `aeon_validator_pool`
  - handles: staking, epochs, NFT issuance
- `KeeperConsensus`
  - var: `aeon_keeper_vote`
  - handles: memory submission, voting, quorum
- `MemoryLedger`
  - var: `aeon_memory_log`
  - handles: memory anchoring, hash tracking, submission logging
- `CodexGuardian`
  - var: `aeon_codex_protect`
  - handles: Codex hashing, update validation, Ethos token safeguard
- `EthosFailSafe`
  - var: `aeon_ethos_token`
  - handles: token burn trigger, VPC failover, internal recursion audit

## 3. Directory Structure
```
UniCareOS/
├── build/                  # Build scripts and CI
├── config/                 # Genesis, chain config files
├── contracts/              # Smart contracts for Keeper voting and NFT logic
├── core/
│   ├── validator_pool.go
│   ├── keeper_consensus.go
│   ├── memory_ledger.go
│   ├── codex_guardian.go
│   └── ethos_failsafe.go
├── test/                   # Unit and integration tests
├── scripts/                # Utility scripts: integrity, audit, deploy
├── logs/
│   └── unicareos_build_scratchpad.md
└── README.md
```

## 4. Coding Rules / Guardrails
- **Do not invent variable names.** Only use those declared in this file.
- **Do not rename or refactor modules unless explicitly versioned.**
- **Do not use OpenAI-generated speculative CLI or RPC commands.**
- **Do not modify `codex_guardian.go` or `ethos_failsafe.go` unless reviewed.**

## 5. Active Working Log (unicareos_build_scratchpad.md)
> This file stores all name changes, errors, shifts in build logic. You update this manually *after every change* to preserve traceability.

## 6. AI Prompt Guide (Session Standard)
When starting a coding session with AI:
```
"You are contributing to UniCareOS. Use ONLY the module and variable names defined in the reference blueprint. Do not rename, invent, or shift architecture. We are using Go 1.21.x and Avalanche Subnet logic."
```

## 7. Audit Scripts (Optional)
Scripts that could be added later:
- `check_integrity.py`: Cross-reference blueprint vs. source files
- `validate_structure.sh`: Ensure no frozen files are modified

## 8. Git Commit Discipline
- Tag milestones: `v0.1-blueprint`, `v0.2-genesis`, etc.
- Always `git add` + update scratchpad after major architectural changes
