package cos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type stsCredentials struct {
	secretID   string
	secretKey  string
	sessionToken string
	expiresAt  time.Time
}

// credentials returns an STS session, caching it until near expiry. The lazy
// exchange keeps the permanent keys off the wire except during the STS call.
func (c *client) credentials(ctx context.Context) (*stsCredentials, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sts != nil && time.Now().Before(c.sts.expiresAt.Add(-2*time.Minute)) {
		return c.sts, nil
	}
	fetch := c.fetchSts
	if fetch == nil {
		fetch = c.fetchSTS
	}
	creds, err := fetch(ctx)
	if err != nil {
		return nil, err
	}
	c.sts = creds
	return creds, nil
}

// fetchSTS calls GetFederationToken and parses the temp credentials. The
// signature is Tencent Cloud TC3-HMAC-SHA256.
func (c *client) fetchSTS(ctx context.Context) (*stsCredentials, error) {
	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	ts := strconv.FormatInt(now.Unix(), 10)

	// policy limited to the object namespace of this bucket (no prefix) so the
	// temp creds can PUT/GET/DELETE any object under it.
	policyDoc := fmt.Sprintf(
		`{"version":"2.0","statement":[{"action":["name/cos:PutObject","name/cos:GetObject","name/cos:HeadObject","name/cos:DeleteObject"],"effect":"allow","resource":"qcs::cos:%s:uid/1252305156:%s/*"}]}`,
		c.region, c.bucket)
	payload := fmt.Sprintf(
		`{"Name":"disapp","Policy":%s,"DurationSeconds":1800}`,
		strconv.Quote(policyDoc))

	hashedPayload := sha256Hex([]byte(payload))
	canonicalHeaders := "content-type:application/json\nhost:sts.tencentcloudapi.com\n"
	signedHeaders := "content-type;host"
	canonicalRequest := "POST\n/\n" + "\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + hashedPayload
	scope := date + "/sts/tc3_request"
	stringToSign := "TC3-HMAC-SHA256\n" + ts + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))

	secretDate := hmacSHA256Bytes([]byte("TC3"+c.secretKey), date)
	secretService := hmacSHA256Bytes(secretDate, "sts")
	secretSigning := hmacSHA256Bytes(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256Bytes(secretSigning, stringToSign))
	authorization := "TC3-HMAC-SHA256 Credential=" + c.secretID + "/" + scope +
		", SignedHeaders=" + signedHeaders + ", Signature=" + signature

	debugf(c, "STS canonicalRequest:\n%s", canonicalRequest)
	debugf(c, "STS stringToSign:\n%s", stringToSign)
	debugf(c, "STS payload: %s", payload)
	debugf(c, "STS authorization: %s", authorization)
	debugf(c, "STS secretID: %s", c.secretID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://sts.tencentcloudapi.com/", bytes.NewReader([]byte(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "sts.tencentcloudapi.com")
	req.Header.Set("X-TC-Action", "GetFederationToken")
	req.Header.Set("X-TC-Version", "2018-08-13")
	req.Header.Set("X-TC-Region", c.region)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(now.Unix(), 10))
	req.Header.Set("Authorization", authorization)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Response struct {
			Credentials struct {
				TmpSecretID  string `json:"TmpSecretId"`
				TmpSecretKey string `json:"TmpSecretKey"`
				Token        string `json:"Token"`
			} `json:"Credentials"`
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("cos sts decode: %w", err)
	}
	if cResp := out.Response; cResp.Credentials.TmpSecretID == "" {
		if cResp.Error.Code == "" {
			return nil, fmt.Errorf("cos sts: empty credentials (http %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("cos sts: %s: %s", cResp.Error.Code, cResp.Error.Message)
	}
	return &stsCredentials{
		secretID:     out.Response.Credentials.TmpSecretID,
		secretKey:    out.Response.Credentials.TmpSecretKey,
		sessionToken: out.Response.Credentials.Token,
		expiresAt:    now.Add(30 * time.Minute),
	}, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256Bytes(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}

// debugf logs the STS signing intermediates when STS_DEBUG=1. Useful on a
// machine with outbound network when STS rejects the request.
func debugf(c *client, format string, args ...any) {
	if os.Getenv("STS_DEBUG") == "1" && c != nil {
		log.Printf("STS "+format, args...)
	}
}