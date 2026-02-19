# 🐹 Go Examples

> Last updated: 2026-02-19

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

## Bootstrap

Prerequisites:

- Go 1.25+

Recommended environment variables:

```bash
export BASE_URL="http://localhost:8080"
export CLIENT_ID="<client-id>"
export CLIENT_SECRET="<client-secret>"
```

```go
package main

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

var baseURL = envOrDefault("BASE_URL", "http://localhost:8080")

func main() {
    token, err := issueToken(
        envOrDefault("CLIENT_ID", "<client-id>"),
        envOrDefault("CLIENT_SECRET", "<client-secret>"),
    )
    if err != nil {
        panic(err)
    }

    if err := createSecret(token, "/app/prod/go-example", "go-secret-value"); err != nil {
        panic(err)
    }

    ciphertext, err := transitEncrypt(token, "go-pii", "john@example.com")
    if err != nil {
        panic(err)
    }

    plaintextB64, err := transitDecrypt(token, "go-pii", ciphertext)
    if err != nil {
        panic(err)
    }

    if plaintextB64 != base64.StdEncoding.EncodeToString([]byte("john@example.com")) {
        panic("round-trip verification failed")
    }

    decoded, err := base64.StdEncoding.DecodeString(plaintextB64)
    if err != nil {
        panic(err)
    }
    fmt.Println("decrypted value:", string(decoded))

    fmt.Println("Transit round-trip verified")
}

func envOrDefault(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}

func issueToken(clientID, clientSecret string) (string, error) {
    body := map[string]string{"client_id": clientID, "client_secret": clientSecret}
    data, _ := json.Marshal(body)

    resp, err := http.Post(baseURL+"/v1/token", "application/json", bytes.NewReader(data))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    raw, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusCreated {
        return "", fmt.Errorf("token status=%d body=%s", resp.StatusCode, string(raw))
    }

    var out struct{ Token string `json:"token"` }
    if err := json.Unmarshal(raw, &out); err != nil {
        return "", err
    }
    return out.Token, nil
}

func createSecret(token, path, value string) error {
    payload := map[string]string{"value": base64.StdEncoding.EncodeToString([]byte(value))}
    data, _ := json.Marshal(payload)

    req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/secrets"+path, bytes.NewReader(data))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 300 {
        raw, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("create secret status=%d body=%s", resp.StatusCode, string(raw))
    }
    return nil
}

func transitEncrypt(token, keyName, plaintext string) (string, error) {
    _ = createTransitKey(token, keyName)

    payload := map[string]string{"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext))}
    data, _ := json.Marshal(payload)
    req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/transit/keys/"+keyName+"/encrypt", bytes.NewReader(data))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    raw, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("encrypt status=%d body=%s", resp.StatusCode, string(raw))
    }

    var out struct{ Ciphertext string `json:"ciphertext"` }
    if err := json.Unmarshal(raw, &out); err != nil {
        return "", err
    }
    // For transit decrypt, pass ciphertext exactly as returned by encrypt: "<version>:<base64-ciphertext>".
    return out.Ciphertext, nil
}

func transitDecrypt(token, keyName, ciphertext string) (string, error) {
    payload := map[string]string{"ciphertext": ciphertext}
    data, _ := json.Marshal(payload)
    req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/transit/keys/"+keyName+"/decrypt", bytes.NewReader(data))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    raw, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("decrypt status=%d body=%s", resp.StatusCode, string(raw))
    }

    var out struct{ Plaintext string `json:"plaintext"` }
    if err := json.Unmarshal(raw, &out); err != nil {
        return "", err
    }
    return out.Plaintext, nil
}

func createTransitKey(token, keyName string) error {
    payload := map[string]string{"name": keyName, "algorithm": "aes-gcm"}
    data, _ := json.Marshal(payload)

    req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/transit/keys", bytes.NewReader(data))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}
```

## Tokenization Quick Snippet

```go
func tokenizationFlow(token string) error {
    _ = createTokenizationKey(token, "go-tokenization")

    tokenValue, err := tokenize(token, "go-tokenization", "sensitive-value")
    if err != nil {
        return err
    }

    plaintextB64, err := detokenize(token, tokenValue)
    if err != nil {
        return err
    }

    expected := base64.StdEncoding.EncodeToString([]byte("sensitive-value"))
    if plaintextB64 != expected {
        return fmt.Errorf("tokenization round-trip verification failed")
    }

    return nil
}
```

Deterministic caveat:

- Keys configured as deterministic can emit the same token for the same plaintext under the same active key.
- Use deterministic mode only when your workflow requires equality matching.

## Common Mistakes

- Posting raw strings instead of base64-encoded fields for secrets/transit payloads
- Generating decrypt `ciphertext` from local assumptions instead of encrypt response
- Missing bearer token header on one request in a multi-step flow
- Ignoring `409 Conflict` on transit create and not switching to rotate logic
- Sending tokenization token in URL path instead of JSON body for `detokenize`, `validate`, and `revoke`

## See also

- [Authentication API](../api/authentication.md)
- [Secrets API](../api/secrets.md)
- [Transit API](../api/transit.md)
- [Tokenization API](../api/tokenization.md)
- [Response shapes](../api/response-shapes.md)
