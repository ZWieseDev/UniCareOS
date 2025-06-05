package cmd

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"github.com/spf13/pflag"
	"github.com/spf13/cobra"
)

// Response structure for /api/v1/get-lineage
type LineageResponse struct {
	DocLineage     []string      `json:"docLineage"`
	Event          interface{}   `json:"event"`
	EventID        string        `json:"eventId"`
	RecordID       string        `json:"recordId"`
	RevisionOf     string        `json:"revisionOf"`
	RevisionReason string        `json:"revisionReason"`
}

var (
	lineageEventID string
	lineageServer  string
	lineageInsecure bool
	lineageOutput   string
	lineageFull     bool
	lineageToken    string
	lineageApiKey   string
) // --output flag: 'text' (default) or 'json', --full for all metadata (still redacted)


var viewRevisionsCmd = &cobra.Command{
	Use:   "view-revisions",
	Short: "View the full revision lineage for a medical record event",
	Long: `View the full revision lineage for a medical record event in UniCareOS.

Usage:
  unicare-cli view-revisions --eventId=<eventId> [flags]

Flags:
  --eventId           (required) EventID to query
  --server            API server base URL (default: https://localhost:8080)
  --insecure          Skip TLS certificate verification (for local/dev)
  --output            Output format: 'text' (default) or 'json'
  --full              Show all metadata fields (still redacted for PHI/PII)
  --eventType         Filter by event type (e.g., correction, creation)
  --from              Filter by start date (YYYY-MM-DD)
  --to                Filter by end date (YYYY-MM-DD)
  --authorValidator   Filter by author validator (hex)

Examples:
  # Basic usage
  unicare-cli view-revisions --eventId=abcd1234

  # With token authentication
  unicare-cli view-revisions --eventId=abcd1234 --token=your-secure-token-here

  # With API key authentication
  unicare-cli view-revisions --eventId=abcd1234 --api-key=dummy-api-key

  # Filter by event type and date range
  unicare-cli view-revisions --eventId=abcd1234 --eventType=correction --from=2025-01-01 --to=2025-06-01

  # Show all metadata (still redacted), output as JSON
  unicare-cli view-revisions --eventId=abcd1234 --output=json --full

Troubleshooting:
- If you see TLS errors, use --insecure for local/dev servers.
- If you get 'Error: --eventId is required', make sure to supply a valid EventID.
- Only audit/admin metadata is shown; all PHI/PII is redacted or omitted for compliance.
- For backend errors or missing data, check server logs or ensure your API server is running.
`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[Compliance Notice] All queries are logged for audit purposes. Only audit/admin metadata is shown; all PHI/PII is redacted.")
		if lineageEventID == "" {
			fmt.Println("Error: --eventId is required. Use --eventId=<eventId> to specify the record.")
			cmd.Usage()
			os.Exit(1)
		}

		// Check for unknown flags (fallback, since Cobra will error for truly unknown flags)
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed && f.Value.String() != f.DefValue {
				fmt.Printf("Warning: Unknown or unused flag --%s\n", f.Name)
			}
		})
		if lineageServer == "" {
			lineageServer = "https://localhost:8080" // default for local dev
		}
		eventType, _ := cmd.Flags().GetString("eventType")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		authorValidator, _ := cmd.Flags().GetString("authorValidator")

		params := fmt.Sprintf("eventId=%s", lineageEventID)
		if eventType != "" {
			params += "&eventType=" + eventType
		}
		if from != "" {
			params += "&from=" + from
		}
		if to != "" {
			params += "&to=" + to
		}
		if authorValidator != "" {
			params += "&authorValidator=" + authorValidator
		}
		url := fmt.Sprintf("%s/api/v1/get-lineage?%s", lineageServer, params)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: lineageInsecure},
		}
		client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

		// Prepare request with optional auth headers
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Error building request: %v\n", err)
			os.Exit(1)
		}
		if lineageToken != "" {
			req.Header.Set("Authorization", "Bearer "+lineageToken)
		}
		if lineageApiKey != "" {
			req.Header.Set("X-API-Key", lineageApiKey)
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error connecting to server: %v\n", err)
			fmt.Println("Troubleshooting: Check your --server URL, network connection, or use --insecure for local/dev servers.")
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("Server returned status %d\n", resp.StatusCode)
			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				fmt.Println("Authorization error: Check your credentials or backend access policies.")
			}
			fmt.Println("Troubleshooting: Check backend logs for more details.")
			os.Exit(1)
		}

		var lineageResp LineageResponse
		if err := json.NewDecoder(resp.Body).Decode(&lineageResp); err != nil {
			fmt.Printf("Error decoding response: %v\n", err)
			os.Exit(1)
		}

		if lineageOutput == "json" {

			// Convert EventID and AuthorValidator to hex strings if present
			eventMap, ok := lineageResp.Event.(map[string]interface{})
			if ok {
				if eid, exists := eventMap["EventID"]; exists {
					if arr, ok := eid.([]interface{}); ok {
						eventMap["EventID"] = intArrayToHex(arr)
					}
				}
				if av, exists := eventMap["AuthorValidator"]; exists {
					if arr, ok := av.([]interface{}); ok {
						eventMap["AuthorValidator"] = intArrayToHex(arr)
					}
				}
				// Redact PII/PHI fields unless --full is set (but even with --full, redact actual values)
				fieldsToRedact := []string{"PatientID", "ProviderID", "Description", "revisionReason", "recordId"}
				for _, f := range fieldsToRedact {
					if _, exists := eventMap[f]; exists {
						if !lineageFull {
							delete(eventMap, f)
						} else {
							eventMap[f] = "[REDACTED]"
						}
					}
				}
				lineageResp.Event = eventMap
			}
			jsonBytes, err := json.MarshalIndent(lineageResp, "", "  ")
			if err != nil {
				fmt.Printf("Error encoding JSON output: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(jsonBytes))
			return
		}

		// Default: text output
		if lineageOutput != "text" && lineageOutput != "json" {
			fmt.Printf("Warning: Unknown output format '%s'. Defaulting to 'text'.\n", lineageOutput)
		}
		fmt.Println("Revision Lineage:")
		for i, eid := range lineageResp.DocLineage {
			fmt.Printf("  %d. %s\n", i+1, eid)
		}
		fmt.Println("---")
		fmt.Printf("EventID: %s\nRecordID: %s\nRevisionOf: %s\nRevisionReason: %s\n",
			lineageResp.EventID, lineageResp.RecordID, lineageResp.RevisionOf, lineageResp.RevisionReason)

		// Print Event Metadata fields as hex
		fmt.Println("Event Metadata:")
		eventMap, ok := lineageResp.Event.(map[string]interface{})
		if ok {
			fieldsToRedact := []string{"PatientID", "ProviderID", "Description", "revisionReason", "recordId"}
			for _, f := range fieldsToRedact {
				if _, exists := eventMap[f]; exists {
					if !lineageFull {
						continue // do not show
					} else {
						fmt.Printf("  %s: [REDACTED]\n", f)
					}
				}
			}
			if eid, exists := eventMap["EventID"]; exists {
				fmt.Println("  EventID (hex):", intArrayToHex(eid.([]interface{})))
			}
			if av, exists := eventMap["AuthorValidator"]; exists {
				fmt.Println("  AuthorValidator (hex):", intArrayToHex(av.([]interface{})))
			}
		}

	},
}

// Helper: convert []interface{} (from JSON array) to hex string
func intArrayToHex(arr []interface{}) string {
	b := make([]byte, len(arr))
	for i, v := range arr {
		b[i] = byte(v.(float64)) // JSON numbers are float64
	}
	return fmt.Sprintf("%x", b)
}

func init() {
	viewRevisionsCmd.Flags().StringVar(&lineageEventID, "eventId", "", "EventID to query (required)")
	viewRevisionsCmd.Flags().StringVar(&lineageServer, "server", "https://localhost:8080", "API server base URL")
	viewRevisionsCmd.Flags().BoolVar(&lineageInsecure, "insecure", false, "Skip TLS certificate verification (for local/dev)")
	viewRevisionsCmd.Flags().StringVar(&lineageOutput, "output", "text", "Output format: 'text' (default) or 'json'")
	viewRevisionsCmd.Flags().BoolVar(&lineageFull, "full", false, "Show all metadata fields (still redacted for PHI/PII)")
	viewRevisionsCmd.Flags().StringVar(&lineageToken, "token", "", "JWT token for Authorization header (optional)")
	viewRevisionsCmd.Flags().StringVar(&lineageApiKey, "api-key", "", "API key for X-API-Key header (optional)")
	viewRevisionsCmd.Flags().String("eventType", "", "Filter by event type (e.g., correction, creation)")
	viewRevisionsCmd.Flags().String("from", "", "Filter by start date (YYYY-MM-DD)")
	viewRevisionsCmd.Flags().String("to", "", "Filter by end date (YYYY-MM-DD)")
	viewRevisionsCmd.Flags().String("authorValidator", "", "Filter by author validator (hex)")

	rootCmd.AddCommand(viewRevisionsCmd)
}
