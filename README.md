# UniCareOS - Blockchain-Powered Healthcare Data Exchange

![UniCareOS Banner](https://via.placeholder.com/1200x400/2a3b4d/ffffff?text=UniCareOS+Blockchain+Healthcare)

## 🏥 Project Overview

UniCareOS is a purpose-built, permissioned ledger designed to modernize how healthcare providers exchange and audit medical records. We replace expensive, error-prone point-to-point feeds (CDs, faxes, VPN-based HL7 links) with a single federated network—no tokens, no speculative fees—just a BAA-ready, AWS-hosted architecture that delivers:

- **Sub-minute scan-to-specialist turnaround** for MRIs, lab reports, and DICOM images
- **Immutable audit trails** for every record write, consent change, and emergency-access event
- **Patient-centric consent controls** via DID-based wallets (QR scan + on-chain signature)

UniCareOS isn't "blockchain for crypto." It's a HIPAA-safe, token-free distributed ledger that integrates with existing EHR infrastructures, giving healthcare providers a faster, more secure way to share medical data.

## 🎯 Key Features

### 🔄 Real-Time Record Exchange
- Replace multi-day CD/fax/PACS transfers with under-60-second digital delivery
- Built-in HL7 v2.x & FHIR adapters for seamless EHR integration

### 🔒 End-to-End Security
- Encrypted off-chain storage with AWS KMS envelope encryption
- On-chain metadata and consent management
- DID-based identity for patients and providers

### 📊 Comprehensive Portals

#### Patient Portal
![Patient Portal](./images/Patient%20Portal%20Preview.png)
*View personal records, manage consents, and track access history*

#### Facility Portal
![Facility Portal](./images/Facility%20Portal.png)
*Submit new records, manage patient consents, and handle emergency access requests*

#### Audit Portal
![Audit Portal](./images/Audit%20Portal.png)
*Track all system activities with immutable, timestamped records*

#### Blockchain Explorer
![Blockchain Explorer](./images/Explorer%20Homepage.png)
*Monitor transactions, blocks, and network health in real-time*

## 🏗️ Architecture

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
                        │ │  • Block Production (3s)   │
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

## 🚀 Getting Started

### Prerequisites

- Go 1.20+
- Node.js 16+ (for Explorer/UI)
- Docker (for local development)
- AWS Account (for KMS, S3 in production)

### Local Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/ZWieseDev/UniCareOS.git
   cd UniCareOS
   ```

2. **Start LocalStack (S3 & KMS emulation)**
   ```bash
   docker run --rm -it -e SERVICES="s3,kms" -p 4566:4566 localstack/localstack
   ```

3. **Configure AWS CLI for LocalStack**
   ```bash
   aws configure set aws_access_key_id test
   aws configure set aws_secret_access_key test
   aws configure set region us-east-1
   aws configure set endpoint_url http://localhost:4566
   ```

4. **Create S3 bucket & KMS key**
   ```bash
   aws --endpoint-url=http://localhost:4566 kms create-key --description "UniCareOS Local DEK"
   aws --endpoint-url=http://localhost:4566 s3api create-bucket --bucket unicareos-records
   ```

5. **Build and start the Validator Node**
   ```bash
   cd cmd/unicare-node
   go build -o unicare-node .
   ./unicare-node --config=../../config/local-config.yaml
   ```

6. **Start the Explorer UI**
   ```bash
   cd ../../explorer
   npm install
   npm run dev
   ```

Visit `http://localhost:3000` to access the Explorer interface.

## 📚 Documentation

For detailed documentation, including API references and deployment guides, please visit our [Wiki](https://github.com/ZWieseDev/UniCareOS/wiki).

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details on how to get started.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Contact

For inquiries, please contact [resume@zwiese.com](mailto:resume@zwiese.com).

---

<div align="center">
  Made with ❤️ by ZWieseDev | [View on GitHub](https://github.com/ZWieseDev/UniCareOS)
</div>
