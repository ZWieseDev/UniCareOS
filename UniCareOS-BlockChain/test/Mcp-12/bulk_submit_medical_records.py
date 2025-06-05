import requests
import json
import random
import uuid
from base64 import b64encode

# Color codes for terminal output
GREEN = "\033[92m"
RED = "\033[91m"
RESET = "\033[0m"

API_URL = "http://localhost:8080/api/v1/submit-medical-record"
HEADERS = {
    "Authorization": "Bearer your-secure-token-here",
    "Content-Type": "application/json"
}

def random_base64(nbytes=12):
    return b64encode(random.randbytes(nbytes)).decode()

def make_record(valid=True):
    # Generate a valid or slightly invalid record for testing
    record = {
        "recordId": str(uuid.uuid4()),
        "patientId": random_base64() if valid else "not_base64!",
        "patientDID": "did:example:123456789abcdefghi",
        "providerId": "encrypted-provider-id",
        "docHash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        "schemaVersion": "1.0",
        "recordType": "lab_result",
        "issuedAt": "2025-05-25T15:00:00Z",
        "signedBy": "prov123",
        "retentionPolicy": "standard",
        "encryptionContext": {
            "algorithm": "AES-GCM",
            "iv": "abcdefghijklmnop1234",
            "tag": "ZYXWVUTSRQPONMLK9876"
        },
        "consentStatus": "granted",
        "dataProvenance": "hospital-system",
        "payloadSignature": random_base64(16)
    }
    return {
        "record": record,
        "walletAddress": "prov123"
    }

def submit_record(payload):
    resp = requests.post(API_URL, headers=HEADERS, data=json.dumps(payload))
    return resp

def main():
    # Try a mix of valid and invalid records
    for i in range(10):
        valid = (i % 2 == 0)
        record = make_record(valid=valid)
        resp = submit_record(record)
        try:
            out = resp.json()
        except Exception:
            out = resp.text
        if resp.status_code == 200:
            print(f"{GREEN}[OK]{RESET} Record {i+1} submitted successfully: {out}")
        else:
            print(f"{RED}[FAIL]{RESET} Record {i+1} failed: {out}")

if __name__ == "__main__":
    main()
