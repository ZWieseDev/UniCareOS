Project Overview
UniCareOS is a purposeâ€built, permissioned ledger designed to modernize how healthcare providers exchange and audit medical records. We replaced expensive, errorâ€prone pointâ€toâ€point feeds (CDs, faxes, VPNâ€based HL7 links) with a single federated networkâ€”no tokens, no speculative feesâ€”just a BAAâ€ready, AWSâ€hosted architecture that delivers:

Sub-minute scanâ€toâ€specialist turnaround (MRI, lab reports, DICOM images).

Immutable audit trails for every record write, consent change, and emergencyâ€access event.

Patientâ€centric consent controls via DIDâ€based wallets (QR scan + onâ€chain signature).

UniCareOS isnâ€™t â€œblockchain for crypto.â€ Itâ€™s a HIPAAâ€safe, tokenâ€free distributed ledger that slots into existing EHR infrastructures, giving hospitals and clinics a faster, more secure way to share medical dataâ€”and patients a simple way to grant or revoke permission in real time.

Key Goals
Realâ€Time Record Exchange

Replace multi-day CD/fax/PACS transfers with underâ€60â€second digital delivery.

Endâ€toâ€End Auditable History

Every action (record submission, consent grant/revoke, break-glass access) is cryptographically timestamped onâ€chain.

Patient-First Consent Model

Patients hold their own DID in a secure wallet or smart badge. They scan a QR code to grant/revoke accessâ€”no paperwork, no middleman.

Plug-and-Play Integration

Builtâ€in HL7 v2.x & FHIR adapter lets EHRs connect with a single configurationâ€”no brittle, custom VPN feeds.

Predictable, Token-Free Costs

All infrastructure runs on AWS EC2, S3 & KMS under a BAAâ€”no perâ€transaction gas fees, no price volatility.

Core Features
Permissioned Validator Network

Hospitals, labs and imaging centers run small, highly available nodes in AWS (or on-prem), each maintaining a copy of the ledger.

Liquid Contracts (Onâ€Chain Governance)

Smart, zeroâ€downtime rules enforce bans (rate limits, peer exclusions), consent logic, and emergency access directly within the ledger.

Encrypted Offâ€Chain Storage

Medical payloads (PDFs, DICOM, images) are envelopeâ€encrypted with AWS KMS and stored in S3. Only the hash and metadata live onâ€chain.

DIDâ€Based Identity & Wallets

Each patient and staff member has a decentralized identifier (DID). Hardware badges or secure mobile wallets sign transactions for onâ€chain actions.

HL7 v2.x & FHIR Ingestion

Native adapter ingests legacy HL7 feeds or FHIR API calls, transforms them to SubmitMedicalRecordTx, and publishes on the ledger.

Consent & Emergency Access Flows

Patients scan a QR to grant or revoke permissions. In break-glass scenarios, authorized clinicians can trigger an onâ€chain emergencyâ€access event (fully audited).

Explorer & Portals

Webâ€based interfaces for:

Patient Portal: View personal records, manage consents, audit history.

Facility Portal: Providers submit records, view patients under consent, request emergency access.

Audit Logs: Compliance officers drill into every event, export CSV/JSON, and verify HIPAA requirements.

Architecture
pgsql
Copy
Edit
                          â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
                          â”‚     External Users       â”‚
                          â”‚ â”Œâ”€â”€â”€â”€â”€â”€â”   â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â” â”‚
   QR-Scan & DID Sign   â”‚ â”‚Patientâ”‚   â”‚Provider â”‚ â”‚
                        â”‚ â””â”€â”€â”€â”¬â”€â”€â”˜   â””â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”˜ â”‚
                        â”‚     â–¼              â”‚     â”‚
                        â”‚ â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â” â”‚
   Consent TXs & Reads  â”‚ â”‚    Web Explorer     â”‚ â”‚
                        â”‚ â”‚ (React / Next.js)   â”‚ â”‚
                        â”‚ â””â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜ â”‚
                        â”‚   â”‚                   â”‚  â”‚
                        â”‚   â–¼                   â”‚  â”‚
â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”  â”‚ â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â” â”‚
â”‚      Wallets       â”‚â—„â”€â”¼â”€â”‚  REST API / GraphQL     â”‚ â”‚
â”‚ (Mobile & Badges)  â”‚   â”‚ (Node.js / Go Gateway)  â”‚ â”‚
â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜   â””â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜ â”‚
                        â”‚   â”‚                       â”‚
                        â”‚   â–¼                       â”‚
                        â”‚ â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
                        â”‚ â”‚    Validator Node (Go)     â”‚
                        â”‚ â”‚  â€¢ P2P Gossip & Mempool    â”‚
                        â”‚ â”‚  â€¢ Block Production (3 s)  â”‚
                        â”‚ â”‚  â€¢ Liquid Contracts (Wasm) â”‚
                        â”‚ â”‚  â€¢ HL7/FHIR Adapter        â”‚
                        â”‚ â””â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
                        â”‚   â”‚
                        â”‚   â–¼
 â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”  â”‚ â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”  â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
 â”‚ AWS KMS (DEKs)    â”‚  â”‚ â”‚ S3 (Encrypted Record Blobs)   â”‚  â”‚ Monitoring & Logs   â”‚
 â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜  â”‚ â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜  â”‚ (CloudWatch, Grafana) â”‚
                        â”‚
                        â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
Validator Node (Go)

P2P gossip, mempool, block assembly, signature validation, and onâ€chain â€œliquid contractsâ€ for consent, emergency access, and bans.

HL7 v2.x and FHIR adapter modules ingest legacy hospital data (PDF, DICOM) and emit SubmitMedicalRecordTx.

REST API / GraphQL Gateway

Serves patient, provider, and auditor portals. Validates DID signatures for write endpoints.

Web Explorer (React / Next.js)

Singleâ€page app with Roleâ€Based Access:

Patient Portal (view records, manage consents).

Facility Portal (submit new records, view roster).

Audit Logs (export all events, filter by DID, date, eventType).

AWS Services

KMS: Envelopeâ€encrypt DEKs for each record.

S3: Store encrypted record blobs.

CloudWatch / Grafana: Monitor node health, TPS, orphan rate, mempool depth.

Getting Started
Prerequisites
Go 1.20+

Node.js 16+ (for the Explorer/UI)

Docker (for local S3/KMS emulation or quick prototyping)

An AWS account with:

IAM permissions for S3 and KMS (create/read/write usage)

(Optional) AWS CLI configured for staging/production deployment

Clone & Build
bash
Copy
Edit
# 1. Clone the monorepo
git clone https://github.com/unicareos/core-chain.git
cd core-chain

# 2. Build the Validator Node
cd cmd/unicare-node
go build -o unicare-node .

# 3. Build the Explorer UI
cd ../../explorer
npm install
npm run build
Run a Local Node (Development Mode)
For local testing, you can run a Validator Node against the LocalStack S3/KMS emulation:

bash
Copy
Edit
# 1. Start LocalStack (S3 & KMS)
docker run --rm -it \
  -e SERVICES="s3,kms" \
  -p 4566:4566 \
  localstack/localstack

# 2. Configure AWS CLI to point at LocalStack:
aws configure set aws_access_key_id test
aws configure set aws_secret_access_key test
aws configure set region us-east-1
aws configure set endpoint_url http://localhost:4566

# 3. Create an S3 bucket & KMS key in LocalStack
aws --endpoint-url=http://localhost:4566 kms create-key --description "UniCareOS Local DEK"  
aws --endpoint-url=http://localhost:4566 s3api create-bucket --bucket unicareos-records

# 4. Start the Node (uses config/local-config.yaml)
cd cmd/unicare-node
./unicare-node --config=../../config/local-config.yaml
The node will:

Listen on localhost:8080 for REST API.

Gossip on localhost:26656 (P2P port) if you spin up multiple local nodes.

Store blocks & state in ~/.unicareos/data by default.

Using the Explorer
Once the node is running locally, launch the Explorer:

bash
Copy
Edit
cd /path/to/unicareos/explorer
npm run dev
Open your browser at http://localhost:3000. Youâ€™ll see:

Patient Portal (/patient)

â€œConnect Walletâ€ button to unlock your DID.

Timeline of your Medical Records (lab results, imaging, prescriptions).

â€œManage Consentsâ€ section to grant/revoke access to other DIDs.

Medical Facility (Provider) Portal (/facility)

â€œLogin as Providerâ€ (scan badge or enter private key).

Search bar to lookup Patient DID.

â€œSubmit Recordâ€ form (file upload + record type).

â€œEmergency Accessâ€ button for break-glass scenarios.

Audit Logs (/audit)

Filters for eventType (RecordSubmitted, ConsentGranted, EmergencyAccess, Ban).

Search by patientDid, accessorDid, operatorDid, or time range.

â€œExport Logsâ€ button to download CSV/JSON of filtered results.

Integration & Extensibility
HL7 / FHIR Adapter
The Validator Node includes a builtâ€in adapter that listens on localhost:2575 (HL7 v2.x MLLP) or localhost:2755 (FHIR REST).

To point an EHR to your node, configure its HL7 interface to send ADT/ORU messages to host:2575 over TCP (port 2575), or set your FHIR base URL to http://<node-host>:2755/fhir.

Adapter logic will:

Parse incoming message (e.g., lab result PDF embedded in an OBX segment or FHIR DiagnosticReport).

Upload the binary to S3 (encrypted).

Fire a SubmitMedicalRecordTx onâ€chain with the resulting S3 URL, metadata, and operatorDid.

Wallet & DID Support
We follow W3C DID standards. Each userâ€™s DID Document (public key, endpoints) is stored in a lightweight onâ€chain registry.

Wallet Options:

Mobile (React Native): Connects via Expo-secure-store; supports biometric unlock.

Web (Browser): Uses WebAuthn / FIDO2 passkeys or a QR scanner for hardware badges.

To extend or swap in a new wallet:

Implement the SignTx(payload: JSON) â†’ signature interface and inject it into the REST gatewayâ€™s â€œOperator Signatureâ€ middleware.

Security & Compliance
HIPAA-Compliant Encryption

All PHI is encrypted off-chain using envelope encryption (AWS KMS CMK â†’ per-record DEK).

Only encrypted blobs reside in S3; the on-chain ledger stores only SHA-256 hashes, metadata, and consent events.

Liquid Contracts for Consent & Emergency Access

Consent rules (ConsentGrantLC, ConsentRevokeLC) are on-chain Wasm contracts enforcing who can read each record.

Break-glass (EmergencyAccessLC) events are fully logged, multi-sig protected, and require justification.

Key Recovery & Guardian Attestation

Lost wallet keys can be recovered via guardians (family members, designated clinicians) or a secure â€œForgot Keyâ€ flow integrated with AWS Secrets Manager.

Audit-Ready Reports

Compliance officers can filter and export any on-chain event (RecordSubmitted, ConsentGranted, EmergencyAccess, Ban) by DID, timestamp, or eventType.

Network Isolation & Monitoring

All validator nodes run in a dedicated VPC with restricted security groups.

CloudWatch metrics and Grafana dashboards track TPS, orphan rate, mempool depth, and node health.

Contributing
We welcome contributions! To get started:

Fork this repository and create a new branch (feature/my-feature).

Follow our MCP Spec Process to draft a new module or feature spec.

Open a Pull Request with your code and reference the MCP spec.

All PRs require passing tests, CodeQL scanning, and at least one review from a maintainer.

Please read and adhere to our Code of Conduct.

Roadmap
v1.0 (Current)

Core consensus, mempool, gossip, block production (3 s).

HL7 v2.x adapter, FHIR ingest, SubmitMedicalRecordTx, on-chain consent contracts.

Basic Explorer with Patient, Facility, and Audit portals.

v1.1

Multi-node AWS auto-scaling, formal APN onboarding.

Mobile React Native Wallet with NFC badge support.

Extended â€œbreak-glassâ€ attestation flows and multi-sig redaction.

v1.2+

Sharding support per hospital group (federated shards).

Plug-in marketplace for custom â€œliquid contractsâ€ (e.g., insurance pre-auth).

End-to-end HIPAA audit with third-party pen-test & compliance certification.


