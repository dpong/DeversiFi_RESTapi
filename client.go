package dvfapi

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/crypto"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const ENDPOINT = "https://api.deversifi.com"

// Please do not send more than 10 requests per second. Sending requests more frequently will result in HTTP 429 errors.
type Client struct {
	privateKey string
	subaccount string
	HTTPC      *http.Client
}

func New(privateKey, subaccount string) *Client {
	hc := &http.Client{
		Timeout: 10 * time.Second,
	}
	return &Client{
		privateKey: privateKey,
		subaccount: subaccount,
		HTTPC:      hc,
	}
}

func (p *Client) newRequest(method, spath string, body []byte, params *map[string]string) (*http.Request, error) {
	u, _ := url.ParseRequestURI(ENDPOINT)
	u.Path = u.Path + spath
	if params != nil {
		q := u.Query()
		for k, v := range *params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequest(method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	p.Headers(req)

	return req, nil
}

func (c *Client) sendRequest(method, spath string, body []byte, params *map[string]string) (*http.Response, error) {
	req, err := c.newRequest(method, spath, body, params)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTPC.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		return nil, fmt.Errorf("faild to get data. status: %s", res.Status)
	}
	return res, nil
}

func decode(res *http.Response, out interface{}) error {
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	err := json.Unmarshal([]byte(body), &out)
	if err == nil {
		return nil
	}
	return err
}

func responseLog(res *http.Response) string {
	b, _ := httputil.DumpResponse(res, true)
	return string(b)
}
func requestLog(req *http.Request) string {
	b, _ := httputil.DumpRequest(req, true)
	return string(b)
}

func (c *Client) Headers(request *http.Request) {
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json; charset=UTF-8")
}

func SocketEndPointHub(private bool) (endpoint string) {
	switch private {
	case true:
		// pass
	default:
		endpoint = "wss://api.deversifi.com/market-data/ws"

	}
	return endpoint
}

func String2Bytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

type signature struct {
	R *big.Int
	S *big.Int
}

func (p *Client) sign(message string) (string, error) {
	privatekey, err := crypto.HexToECDSA(p.privateKey)
	if err != nil {
		return "", err
	}
	signhash := String2Bytes(message)
	var pubkey ecdsa.PublicKey
	pubkey = privatekey.PublicKey
	r := big.NewInt(0)
	s := big.NewInt(0)
	//io.WriteString(h, "This is a message to be signed and verified by ECDSA!")
	r, s, serr := ecdsa.Sign(rand.Reader, privatekey, signhash)
	if serr != nil {
		return "", serr
	}
	signature := r.Bytes()
	signature = append(signature, s.Bytes()...)
	result := fmt.Sprintf("Signature : %x", signature)
	// Verify
	verifystatus := ecdsa.Verify(&pubkey, signhash, r, s)
	if !verifystatus {
		return "", errors.New("didn't pass verifystatus")
	}
	return result, nil

}

func (p *Client) signAndPublicKey(message string) (string, string, error) {
	privatekey, err := crypto.HexToECDSA(p.privateKey)
	if err != nil {
		return "", "", err
	}
	signhash := String2Bytes(message)
	var pubkey ecdsa.PublicKey
	pubkey = privatekey.PublicKey
	r, s, serr := ecdsa.Sign(rand.Reader, privatekey, signhash)
	if serr != nil {
		return "", "", serr
	}
	signature := r.Bytes()
	signature = append(signature, s.Bytes()...)
	result := fmt.Sprintf("Signature : %x", signature)
	// Verify
	verifystatus := ecdsa.Verify(&pubkey, signhash, r, s)
	if !verifystatus {
		return "", "", errors.New("didn't pass verifystatus")
	}

	publicKey := hex.EncodeToString(append(pubkey.X.Bytes(), pubkey.Y.Bytes()...))
	//publicKey := hex.EncodeToString(pubkey.X.Bytes())
	return result, publicKey, nil
}
