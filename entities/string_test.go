package entities

import "testing"

func TestValidateString(t *testing.T) {
	str := "Hello"
	strLower := "hello"
	strNoUtf := "SSH-2.0-dropbear_2022.83\r\n\u0000\u0000\u0001�\n\u0014*2���`�Bc�T�`���\u0000\u0000\u0000�curve25519-sha256,curve25519-sha256@libssh.org,ecdh-sha2-nistp521,ecdh-sha2-nistp384,ecdh-sha2-nistp256,diffie-hellman-group14-sha256,diffie-hellman-group14-sha1,kexguess2@matt.ucc.asn.au\u0000\u0000\u0000<ssh-ed25519,ecdsa-sha2-nistp521,rsa-sha2-256,ssh-rsa,ssh-dss\u0000\u0000\u00003chacha20-poly1305@openssh.com,aes128-ctr,aes256-ctr\u0000\u0000\u00003chacha20-poly1305@openssh.com,aes128-ctr,aes256-ctr\u0000\u0000\u0000\u0017hmac-sha1,hmac-sha2-256\u0000\u0000\u0000\u0017hmac-sha1,hmac-sha2-256\u0000\u0000\u0000\u0004none\u0000\u0000\u0000\u0004none\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000�ug��R���Y"

	s1, _, err := ValidateString(str, true)
	if strLower != s1 {
		t.Errorf("strings are not equals")
	}

	if err != nil {
		t.Error(err)
	}

	s2, _, err := ValidateString(str, false)
	if str != s2 {
		t.Errorf("strings are not equals")
	}

	if err != nil {
		t.Error(err)
	}

	s3, _, err := ValidateString(strNoUtf, false)
	if strNoUtf == s3 {
		t.Errorf("strings are equals")
	}

	t.Log(s3)

	if err != nil {
		t.Error(err)
	}
}
