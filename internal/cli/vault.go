package cli

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andrey1555251-ux/mindforge/internal/notes"
	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
)

// runVault dispatches `mindforge vault <sub>`.
//
//	mindforge vault encrypt <noteID>   encrypts the body of a note in place
//	mindforge vault decrypt <noteID>   decrypts a previously encrypted note
//	mindforge vault list               lists encrypted notes
func runVault(args []string) error {
	if len(args) == 0 {
		return runVaultList(nil)
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "encrypt", "lock":
		return runVaultEncrypt(rest)
	case "decrypt", "unlock":
		return runVaultDecrypt(rest)
	case "list", "ls":
		return runVaultList(rest)
	}
	return fmt.Errorf("unknown vault subcommand %q", cmd)
}

// vaultPrefix marks a note body as an encrypted blob.  Bumping the
// suffix lets us migrate to a different KDF/cipher later without
// silently mis-decrypting older entries.
const vaultPrefix = "MFVAULT:v1:"

func runVaultEncrypt(args []string) error {
	fs := flag.NewFlagSet("vault encrypt", flag.ContinueOnError)
	passFlag := fs.String("pass", "", "passphrase (or read from MINDFORGE_VAULT_PASS env)")
	args = reorderArgs(args, []string{"pass"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: mindforge vault encrypt <noteID>")
	}
	id, err := atoiPositive(fs.Arg(0))
	if err != nil {
		return err
	}
	pass, err := readPassphrase(*passFlag, "vault passphrase: ", true)
	if err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	return st.Use(func(d *store.Data) error {
		for i := range d.Notes {
			if d.Notes[i].ID != id {
				continue
			}
			if strings.HasPrefix(d.Notes[i].Body, vaultPrefix) {
				return errors.New("note is already encrypted")
			}
			ct, err := encrypt(pass, []byte(d.Notes[i].Body))
			if err != nil {
				return err
			}
			d.Notes[i].Body = vaultPrefix + ct
			fmt.Println(ui.Green("note encrypted"), ui.Dim("#"+itoa(id)))
			return nil
		}
		return fmt.Errorf("note %d not found", id)
	})
}

func runVaultDecrypt(args []string) error {
	fs := flag.NewFlagSet("vault decrypt", flag.ContinueOnError)
	keep := fs.Bool("keep", false, "keep the body encrypted on disk and only print plaintext")
	passFlag := fs.String("pass", "", "passphrase (or read from MINDFORGE_VAULT_PASS env)")
	args = reorderArgs(args, []string{"pass"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: mindforge vault decrypt <noteID> [--keep]")
	}
	id, err := atoiPositive(fs.Arg(0))
	if err != nil {
		return err
	}
	pass, err := readPassphrase(*passFlag, "vault passphrase: ", false)
	if err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	if *keep {
		d, err := st.Snapshot()
		if err != nil {
			return err
		}
		for _, n := range d.Notes {
			if n.ID != id {
				continue
			}
			if !strings.HasPrefix(n.Body, vaultPrefix) {
				return errors.New("note is not encrypted")
			}
			pt, err := decrypt(pass, strings.TrimPrefix(n.Body, vaultPrefix))
			if err != nil {
				return err
			}
			fmt.Println(string(pt))
			return nil
		}
		return fmt.Errorf("note %d not found", id)
	}
	return st.Use(func(d *store.Data) error {
		for i := range d.Notes {
			if d.Notes[i].ID != id {
				continue
			}
			if !strings.HasPrefix(d.Notes[i].Body, vaultPrefix) {
				return errors.New("note is not encrypted")
			}
			pt, err := decrypt(pass, strings.TrimPrefix(d.Notes[i].Body, vaultPrefix))
			if err != nil {
				return err
			}
			d.Notes[i].Body = string(pt)
			fmt.Println(ui.Green("note decrypted"), ui.Dim("#"+itoa(id)))
			return nil
		}
		return fmt.Errorf("note %d not found", id)
	})
}

func runVaultList(args []string) error {
	st, err := open()
	if err != nil {
		return err
	}
	out := notes.Filter(st, notes.FilterOpts{IncludeBody: true, SortByDate: true})
	rows := make([][]string, 0, len(out))
	for _, n := range out {
		if !strings.HasPrefix(n.Body, vaultPrefix) {
			continue
		}
		rows = append(rows, []string{
			ui.Yellow("locked"),
			"#" + itoa(n.ID),
			truncate(n.Title, 40),
			renderTags(n.Tags),
			n.UpdatedAt.Local().Format("Jan 02 15:04"),
		})
	}
	if len(rows) == 0 {
		fmt.Println(ui.Dim("(no encrypted notes)"))
		return nil
	}
	ui.PrintTable(os.Stdout, []string{"", "id", "title", "tags", "updated"}, rows)
	return nil
}

// encrypt seals plaintext with AES-256-GCM using a key derived from
// the passphrase via SHA-256.  The output is base64(nonce||ct) so the
// data file stays ASCII-safe.
func encrypt(pass, plaintext []byte) (string, error) {
	key := sha256.Sum256(pass)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	out := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

func decrypt(pass []byte, blob string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted blob: %w", err)
	}
	key := sha256.Sum256(pass)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, errors.New("blob too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, errors.New("wrong passphrase or corrupted data")
	}
	return pt, nil
}

func atoiPositive(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty id")
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid id %q", s)
		}
		n = n*10 + int(c-'0')
	}
	if n == 0 {
		return 0, fmt.Errorf("invalid id %q", s)
	}
	return n, nil
}

// readPassphrase asks the user for a passphrase.  Order of precedence:
// explicit --pass flag, MINDFORGE_VAULT_PASS env var, then a prompt
// on stderr.  When confirm is true we ask twice and require both to
// match.  We do not attempt to disable terminal echo because portable
// echo-suppression without external deps is fiddly; users who care
// can pipe the passphrase via stdin instead.
func readPassphrase(flagVal, prompt string, confirm bool) ([]byte, error) {
	if flagVal != "" {
		return []byte(flagVal), nil
	}
	if env := os.Getenv("MINDFORGE_VAULT_PASS"); env != "" {
		return []byte(env), nil
	}
	fmt.Fprint(os.Stderr, prompt)
	first, err := readLine(os.Stdin)
	if err != nil {
		return nil, err
	}
	if !confirm {
		return first, nil
	}
	fmt.Fprint(os.Stderr, "confirm passphrase: ")
	second, err := readLine(os.Stdin)
	if err != nil {
		return nil, err
	}
	if string(first) != string(second) {
		return nil, errors.New("passphrases did not match")
	}
	return first, nil
}

func readLine(r io.Reader) ([]byte, error) {
	br := bufio.NewReader(r)
	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return nil, errors.New("empty passphrase")
	}
	return []byte(line), nil
}
