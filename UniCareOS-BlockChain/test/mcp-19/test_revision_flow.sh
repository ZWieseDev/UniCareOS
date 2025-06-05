#!/bin/bash
# Test script for MCP-19: Medical Record Revision Lineage
# This script submits an original record, a first revision, and a second revision,
# then queries the dev inspect endpoint to verify lineage fields.

set -e
API_URL="https://localhost:8080"
HEADER_AUTH="Authorization: Bearer $API_JWT_SECRET"
HEADER_CONTENT="Content-Type: application/json"
ETHOS_TOKEN="eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDgzMjc4NTAsImlhdCI6MTc0ODMyNDI1MCwicm9sZXMiOlsiYWRtaW4iXSwic3ViIjoiMTIzNDU2Nzg5MCJ9.UlxCBu-MZfkKAaNoTifKoSiU151oKJVzdfbKK1oebGbmiMshyHxsSMvsZzpKTOmteQ2AdqxWtIS_5bSYZzyI42iHKBZ4dNLNsABcV7AGSgYduzGzZxO9GbmbDAgJE1AeVVKaP9e1yKh_l6nwMQ0A3Z0GsGJGxDoUBKEldDkxl6CgseTPLB1ym9Qar_gym6BgQ0p3nqthHfGfoQcJQHIpWFP8nX84P1YlWzirx6IvVOusfYyii46fxjRWgfi1hVWTyTIi8jZZ_PQyjkbBJCGNVU8Mhlm5z00w0F1K3ZDY368nFMP4cgRU_VuD9KNzm4u7pLTfKiaYKP7kzN6fRUX2Jw"
HEADER_ETHOS="X-Ethos-Token: $ETHOS_TOKEN"

# Automated multi-step revision lineage test
# List your files in revision order (original, 1st revision, 2nd revision, ...)
FILES=(signed_record.json signed_record_revised_test.json signed_record_revised_test2.json)
declare -a TXIDS

echo "[INFO] Submitting revision chain: ${FILES[*]}"

for i in "${!FILES[@]}"; do
    FILE="${FILES[$i]}"
    if [ "$i" -eq 0 ]; then
        # Submit original as-is
        RESP=$(curl -sk -X POST "$API_URL/api/v1/submit-medical-record" \
            -H "$HEADER_AUTH" -H "$HEADER_CONTENT" -H "$HEADER_ETHOS" \
            --data-binary @"$FILE")
        TXID=$(echo "$RESP" | grep -o '"txId":"[^\"]*"' | cut -d':' -f2 | tr -d '"')
        TXIDS+=("$TXID")
        echo "[STEP $((i+1))] Original submission response: $RESP"
    else
        # For revisions: update revisionOf and docLineage
        PREV_TXID="${TXIDS[$((i-1))]}"
        TMPFILE="tmp_$FILE"
        # Get previous lineage array (if any)
        if [ "$i" -eq 1 ]; then
            # First revision: lineage is just the previous txid
            jq --arg rev "$PREV_TXID" '.revisionOf = $rev | .docLineage = [$rev]' "$FILE" > "$TMPFILE"
        else
            # Subsequent revisions: append to previous lineage
            PREV_LINEAGE=$(jq -c '.docLineage' "tmp_${FILES[$((i-1))]}" 2>/dev/null || echo '[]')
            jq --arg rev "$PREV_TXID" --argjson prevLineage "$PREV_LINEAGE" \
                '.revisionOf = $rev | .docLineage = ($prevLineage + [$rev])' "$FILE" > "$TMPFILE"
        fi
        RESP=$(curl -sk -X POST "$API_URL/api/v1/submit-medical-record" \
            -H "$HEADER_AUTH" -H "$HEADER_CONTENT" -H "$HEADER_ETHOS" \
            --data-binary @"$TMPFILE")
        TXID=$(echo "$RESP" | grep -o '"txId":"[^\"]*"' | cut -d':' -f2 | tr -d '"')
        TXIDS+=("$TXID")
        echo "[STEP $((i+1))] Revision submission response: $RESP"
        rm "$TMPFILE"
    fi
    sleep 2
    # Save the updated file for next iteration lineage extraction
    if [ "$i" -gt 0 ]; then cp "$TMPFILE" "tmp_${FILE}"; fi
    # For original, just copy as-is
    if [ "$i" -eq 0 ]; then cp "$FILE" "tmp_${FILE}"; fi
    
    # Clean up old temp files if not needed
    if [ "$i" -gt 1 ]; then rm -f "tmp_${FILES[$((i-2))]}"; fi

done

# Inspect lineage for each submitted event
for i in "${!TXIDS[@]}"; do
    echo -e "\n[INSPECT] Event $((i+1)) (${FILES[$i]}):"
    curl -sk "$API_URL/dev/inspect_tx?txId=${TXIDS[$i]}" | jq
done

echo -e "\n[RESULT] If lineage is correct, docLineage arrays should grow with each revision."
