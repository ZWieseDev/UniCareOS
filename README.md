# UniCareOS Explorer

> Modernizing healthcare data exchange with a HIPAA-safe, permissioned distributed ledger.

---

## Overview

UniCareOS is a purpose-built, permissioned ledger designed to modernize how healthcare providers exchange and audit medical records. We replace expensive, error-prone point-to-point feeds (CDs, faxes, VPN-based HL7 links) with a single federated network—no tokens, no speculative fees—just a BAA-ready, AWS-hosted architecture that delivers:

- **Sub-minute scan-to-specialist turnaround** (MRI, lab reports, DICOM images)
- **Immutable audit trails** for every record write, consent change, and emergency-access event
- **Patient-centric consent controls** via DID-based wallets (QR scan + on-chain signature)

UniCareOS isn’t “blockchain for crypto.” It’s a HIPAA-safe, token-free distributed ledger that slots into existing EHR infrastructures, giving hospitals and clinics a faster, more secure way to share medical data—and patients a simple way to grant or revoke permission in real time.

---

## Key Goals

- **Real-Time Record Exchange:** Replace multi-day CD/fax/PACS transfers with under-60-second digital delivery.
- **End-to-End Auditable History:** Every action (record submission, consent grant/revoke, break-glass access) is cryptographically timestamped on-chain.
- **Patient-First Consent Model:** Patients hold their own DID in a secure wallet or smart badge. They scan a QR code to grant/revoke access—no paperwork, no middleman.
- **Plug-and-Play Integration:** Built-in HL7 v2.x & FHIR adapter lets EHRs connect with a single configuration—no brittle, custom VPN feeds.
- **Predictable, Token-Free Costs:** All infrastructure runs on AWS EC2, S3 & KMS under a BAA—no per-transaction gas fees, no price volatility.

---

## Core Features

- **Permissioned Validator Network:** Hospitals, labs, and imaging centers run small, highly available nodes in AWS (or on-prem), each maintaining a copy of the ledger.
- **Liquid Contracts (On-Chain Governance):** Smart, zero-downtime rules enforce bans (rate limits, peer exclusions), consent logic, and emergency access directly within the ledger.
- **Encrypted Off-Chain Storage:** Medical payloads (PDFs, DICOM, images) are envelope-encrypted with AWS KMS and stored in S3. Only the hash and metadata live on-chain.
- **DID-Based Identity & Wallets:** Each patient and staff member has a decentralized identifier (DID). Hardware badges or secure mobile wallets sign transactions for on-chain actions.
- **HL7 v2.x & FHIR Ingestion:** Native adapter ingests legacy HL7 feeds or FHIR API calls, transforms them to `SubmitMedicalRecordTx`, and publishes on the ledger.
- **Consent & Emergency Access Flows:** Patients scan a QR to grant or revoke permissions. In break-glass scenarios, authorized clinicians can trigger an on-chain emergency-access event (fully audited).

---

## Explorer & Portals

- **Patient Portal:** View personal records, manage consents, audit history.
- **Facility Portal:** Providers submit records, view patients under consent, request emergency access.
- **Audit Logs:** Compliance officers drill into every event, export CSV/JSON, and verify HIPAA requirements.

---

## Architecture (Text Diagram)

```
                          ┌──────────────────────────┐
                          │     External Users       │
                          │ ┌──────┐   ┌──────────┐ │
   QR-Scan & DID Sign   │ │Patient│   │Provider │ │
                        │ └───┬──┘   └─────┬────┘ │
                        │     ▼              │     │
                        │ ┌─────────────────────┐ │
   Consent TXs & Reads  │ │    Web Explorer     │ │
                        │ │ (React / Next.js)   │ │
                        │ └─┬───────────────────┘ │
                        │   │                   │  │
                        │   ▼                   │  │
┌────────────────────┐  │ ┌─────────────────────────┐ │
│      Wallets       │◄─┼─│  REST API / GraphQL     │ │
│ (Mobile & Badges)  │   │ (Node.js / Go Gateway)  │ │
└────────────────────┘   └─┬───────────────────────┘ │
                        │   │                       │
                        │   ▼                       │
                        │ ┌───────────────────────────┐
                        │ │    Validator Node (Go)     │
                        │ │  • P2P Gossip & Mempool    │
                        │ │  • Block Production (3 s)  │
                        │ │  • Liquid Contracts (Wasm) │
                        │ │  • HL7/FHIR Adapter        │
                        │ └─┬─────────────────────────┘
                        │   │
                        │   ▼
 ┌───────────────────┐  │ ┌───────────────────────────────┐  ┌─────────────────────┐
 │ AWS KMS (DEKs)    │  │ │ S3 (Encrypted Record Blobs)   │  │ Monitoring & Logs   │
 └───────────────────┘  │ └───────────────────────────────┘  │ (CloudWatch, Grafana) │
                        │
                        └─────────────────────────────────────────────────────┘

```

## Screenshots

### Patient Portal
![Patient Portal](./images/Patient%20Portal%20Preview.png)

### Facility Portal
![Facility Portal](./images/Facility%20Portal.png)

### Audit Portal
![Audit Portal](./images/Audit%20Portal.png)

### Explorer Homepage
![Blockchain Explorer](./images/Explorer%20Homepage.png)

---

## Support

For questions or issues, please contact [Your Support Contact Here].

---

*Last updated: June 1, 2025*