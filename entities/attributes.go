package entities

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Attributes represent a collection of various attribute types that can be associated with entities.
type Attributes struct {
	AbaRtn                    *int     `json:"aba-rtn,omitempty" example:"123456789"`
	Adversary                 *string  `json:"adversary,omitempty" example:"APT1"`
	Airport                   *string  `json:"airport-name,omitempty" example:"London"`
	Asn                       *int     `json:"asn,omitempty" example:"12345"`
	Aso                       *string  `json:"aso,omitempty" example:"AS12345"`
	AuthentiHash              *string  `json:"authentihash,omitempty" example:""`
	BankAccountNr             *int     `json:"bank-account-nr,omitempty" example:"987654321"`
	Base64                    *string  `json:"base64,omitempty" example:"SGVsbG8gV29ybGQ="`
	Bic                       *string  `json:"bic,omitempty" example:"BOFAUS3N"`
	Bin                       *int     `json:"bin,omitempty" example:"411111"`
	Breach                    *string  `json:"breach,omitempty" example:"3a7c9d8e-1b2f-4g5h-6j7k-8l9m0n1o2p3q"`
	BreachCount               *int     `json:"breach-count,omitempty" example:"1000000"`
	BreachDate                *string  `json:"breach-date,omitempty" example:"2023-01-15"`
	BreachDescription         *string  `json:"breach-description,omitempty" example:"A major data breach exposing user credentials and personal information"`
	Btc                       *string  `json:"btc,omitempty" example:"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"`
	Category                  *string  `json:"category,omitempty" example:"malware"`
	CcNumber                  *int     `json:"cc-number,omitempty" example:"4111111111111111"`
	CdHash                    *string  `json:"cdhash,omitempty" example:"a3b9e2f18c6d5b4a7d8e9f1a2b3c4d5e"`
	CertificateFingerprint    *string  `json:"certificate-fingerprint,omitempty" example:"5E:FF:56:A2:AF:15:88:DD:F1:D5:B9:A3:E9:BD:1F:48:5F:EE:CB:1E"`
	ChromeExtension           *string  `json:"chrome-extension-id,omitempty" example:"mhjfbmdgcfjbbpaeojofohoefgiehjai"`
	Cidr                      *string  `json:"cidr,omitempty" example:"192.168.1.0/24"`
	City                      *string  `json:"city,omitempty" example:"New York"`
	Command                   *string  `json:"command,omitempty" example:"ping -c 4 example.com"`
	Cookie                    *string  `json:"cookie,omitempty" example:"sessionid=abc123; Path=/; HttpOnly"`
	Country                   *string  `json:"country,omitempty" example:"United States"`
	Cpe                       *string  `json:"cpe,omitempty" example:"cpe:2.3:a:microsoft:windows:10:*:*:*:*:*:*:*"`
	Cve                       *string  `json:"cve,omitempty" example:"CVE-2021-44228"`
	Dash                      *string  `json:"dash,omitempty" example:"XpAy7Zm6aPDPWaJeHHRQ4YECqD1F7bVqhL"`
	Date                      *string  `json:"date,omitempty" example:"2023-05-20"`
	DateOfIssue               *string  `json:"date-of-issue,omitempty" example:"2020-01-01"`
	Datetime                  *string  `json:"datetime,omitempty" example:"2023-05-20T14:30:15.123456789Z"`
	Dkim                      *string  `json:"dkim,omitempty" example:"v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC"`
	DkimSignature             *string  `json:"dkim-signature,omitempty" example:"v=1; a=rsa-sha256; d=example.com; s=selector; c=relaxed/relaxed; q=dns/txt; h=from:to:subject; bh=..."`
	Domain                    *string  `json:"domain,omitempty" example:"example.com"`
	Email                     *string  `json:"email,omitempty" example:"<CAE01+9=7sg@mail.example.com>"`
	EmailAddress              *string  `json:"email-address,omitempty" example:"user@example.com"`
	EmailBody                 *string  `json:"email-body,omitempty" example:"Hello, this is the body of the email message."`
	EmailDisplayName          *string  `json:"email-display-name,omitempty" example:"John Doe"`
	EmailHeader               *string  `json:"email-header,omitempty" example:"From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test Email"`
	EmailMimeBoundary         *string  `json:"email-mime-boundary,omitempty" example:"----=_NextPart_000_0012_01D7A988.9A5F0E30"`
	EmailSubject              *string  `json:"email-subject,omitempty" example:"Important Security Alert"`
	EmailThreadIndex          *string  `json:"email-thread-index,omitempty" example:"AQHSR8AAAA=="`
	EmailXMailer              *string  `json:"email-x-mailer,omitempty" example:"Microsoft Outlook 16.0"`
	Eppn                      *string  `json:"eppn,omitempty" example:"user@university.edu"`
	ExpirationDate            *string  `json:"expiration-date,omitempty" example:"2025-12-31"`
	FacebookProfile           *string  `json:"facebook-profile,omitempty" example:"https://www.facebook.com/username"`
	Ffn                       *string  `json:"ffn,omitempty" example:"ABC123456"`
	File                      *string  `json:"file,omitempty" example:"21a1610ce915d5d5a8ab5b1f5b6d6715cf4f4e3bc0c868352a175279b1881afe"`
	FileData                  *string  `json:"file-data,omitempty" example:"https://example.com/files/sample.exe"`
	Filename                  *string  `json:"filename,omitempty" example:"malware_sample.exe"`
	FilenamePattern           *string  `json:"filename-pattern,omitempty" example:".*\\.exe$"`
	Flight                    *string  `json:"flight,omitempty" example:"AA1234"`
	GitHubOrganization        *string  `json:"github-organization,omitempty" example:"https://github.com/threatwinds"`
	GitHubRepository          *string  `json:"github-repository,omitempty" example:"https://github.com/threatwinds/platform"`
	GitHubUser                *string  `json:"github-user,omitempty" example:"https://github.com/username"`
	Group                     *string  `json:"group,omitempty" example:"APT29"`
	Hex                       *string  `json:"hex,omitempty" example:"0xDEADBEEF"`
	Hostname                  *string  `json:"hostname,omitempty" example:"server01.example.com"`
	Iban                      *string  `json:"iban,omitempty" example:"DE89370400440532013000"`
	IdNumber                  *string  `json:"id-number,omitempty" example:"AB123456"`
	IP                        *string  `json:"ip,omitempty" example:"1.65.1.1"`
	Issuer                    *string  `json:"issuer,omitempty" example:"Department of State"`
	IssuingCountry            *string  `json:"issuing-country,omitempty" example:"United States"`
	Ja3Fingerprint            *string  `json:"ja3-fingerprint,omitempty" example:"e7d705a3286e19ea42f587b344ee6865"`
	JabberId                  *string  `json:"jabber-id,omitempty" example:"user@jabber.org"`
	JarmFingerprint           *string  `json:"jarm-fingerprint,omitempty" example:"27d40d40d29d40d1dc42d43d00041d4689ee210389f4f6b4b5b1b93f92252d"`
	LastAnalysis              *string  `json:"last-analysis,omitempty" example:"2023-06-15T10:30:00.000Z"`
	Latitude                  *float64 `json:"latitude,omitempty" example:"40.7128"`
	Link                      *string  `json:"link,omitempty" example:"https://example.com/reference/doc123"`
	Longitude                 *float64 `json:"longitude,omitempty" example:"-74.0060"`
	MacAddress                *string  `json:"mac-address,omitempty" example:"00:1A:2B:3C:4D:5E"`
	Malware                   *string  `json:"malware,omitempty" example:"pdf dropper agent"`
	MalwareFamily             *string  `json:"malware-family,omitempty" example:"pdf"`
	MalwareSample             *string  `json:"malware-sample,omitempty" example:"https://malware.example.com/samples/trojan.exe"`
	MalwareType               *string  `json:"malware-type,omitempty" example:"dropper"`
	Md5                       *string  `json:"md5,omitempty" example:"d41d8cd98f00b204e9800998ecf8427e"`
	MimeType                  *string  `json:"mime-type,omitempty" example:"application/pdf"`
	MobileAppId               *string  `json:"mobile-app-id,omitempty" example:"com.example.maliciousapp"`
	Os                        *string  `json:"os,omitempty" example:"Windows 10"`
	Passport                  *string  `json:"passport,omitempty" example:"AB1234567"`
	Path                      *string  `json:"path,omitempty" example:"/var/log/suspicious.log"`
	PatternInFile             *string  `json:"pattern-in-file,omitempty" example:"eval\\(base64_decode\\(.*\\)\\)"`
	PatternInMemory           *string  `json:"pattern-in-memory,omitempty" example:"password=[a-zA-Z0-9]{8,}"`
	PatternInTraffic          *string  `json:"pattern-in-traffic,omitempty" example:"User-Agent: Mozilla\\/5\\.0 \\(compatible; MSIE 9\\.0;"`
	Payload                   *string  `json:"payload,omitempty" example:"7a28a1d6ac5b4a7e8c9d0e3f2b1a4c5d6e8f7a9b0c1d2e3f4a5b6c7d8e9f0a1"`
	PgpPrivateKey             *string  `json:"pgp-private-key,omitempty" example:"-----BEGIN PGP PRIVATE KEY BLOCK----- ... -----END PGP PRIVATE KEY BLOCK-----"`
	PgpPublicKey              *string  `json:"pgp-public-key,omitempty" example:"-----BEGIN PGP PUBLIC KEY BLOCK----- ... -----END PGP PUBLIC KEY BLOCK-----"`
	Phone                     *string  `json:"phone,omitempty" example:"+1-555-123-4567"`
	Pnr                       *string  `json:"pnr,omitempty" example:"ABC123"`
	Port                      *string  `json:"port,omitempty" example:"443/tcp"`
	PostalAddress             *string  `json:"postal-address,omitempty" example:"123 Main St, Anytown, CA 12345"`
	Process                   *string  `json:"process,omitempty" example:"svchost.exe"`
	ProcessState              *string  `json:"process-state,omitempty" example:"running"`
	ProfilePhoto              *string  `json:"profile-photo,omitempty" example:"https://example.com/photos/user123.jpg"`
	PRtn                      *string  `json:"prtn,omitempty" example:"1-900-123-4567"`
	RedressNumber             *string  `json:"redress-number,omitempty" example:"987654321"`
	RegKey                    *string  `json:"regkey,omitempty" example:"HKEY_LOCAL_MACHINE\\Software\\Microsoft\\Windows\\CurrentVersion\\Run"`
	Sha1                      *string  `json:"sha1,omitempty" example:"da39a3ee5e6b4b0d3255bfef95601890afd80709"`
	Sha224                    *string  `json:"sha224,omitempty" example:"d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f"`
	Sha256                    *string  `json:"sha256,omitempty" example:"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`
	Sha384                    *string  `json:"sha384,omitempty" example:"38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b"`
	Sha3224                   *string  `json:"sha3-224,omitempty" example:"6b4e03423667dbb73b6e15454f0eb1abd4597f9a1b078e3f5b5a6bc7"`
	Sha3256                   *string  `json:"sha3-256,omitempty" example:"a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a"`
	Sha3384                   *string  `json:"sha3-384,omitempty" example:"0c63a75b845e4f7d01107d852e4c2485c51a50aaaa94fc61995e71bbee983a2ac3713831264adb47fb6bd1e058d5f004"`
	Sha3512                   *string  `json:"sha3-512,omitempty" example:"a69f73cca23a9ac5c8b567dc185a756e97c982164fe25859e0d1dcc1475c80a615b2123af1f5f94c11e3e9402c3ac558f500199d95b6d3e301758586281dcd26"`
	Sha512                    *string  `json:"sha512,omitempty" example:"cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"`
	Sha512224                 *string  `json:"sha512-224,omitempty" example:"6ed0dd02806fa89e25de060c19d3ac86cabb87d6a0ddd05c333b84f4"`
	Sha512256                 *string  `json:"sha512-256,omitempty" example:"c672b8d1ef56ed28ab87c3622c5114069bdd3ad7b8f9737498d0c01ecef0967a"`
	SizeInBytes               *float64 `json:"size-in-bytes,omitempty" example:"1048576"`
	SshBanner                 *string  `json:"ssh-banner,omitempty" example:"SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1"`
	SshFingerprint            *string  `json:"ssh-fingerprint,omitempty" example:"SHA256:uNiVztksCsDhcc0u9e8BujQXVUpKZIDTMczCvj3tD2s"`
	Ssr                       *string  `json:"ssr,omitempty" example:"WCHR"`
	Text                      *string  `json:"text,omitempty" example:"This is a sample text content"`
	Threat                    *string  `json:"threat,omitempty" example:"ransomware"`
	TikTokProfile             *string  `json:"tiktok-profile,omitempty" example:"https://www.tiktok.com/@username"`
	TwitterProfile            *string  `json:"twitter-profile,omitempty" example:"https://twitter.com/username"`
	Url                       *string  `json:"url,omitempty" example:"https://malicious-site.example.com/payload.php"`
	Username                  *string  `json:"username,omitempty" example:"johndoe"`
	Value                     *string  `json:"value,omitempty" example:"sensitive-data-value"`
	Visa                      *string  `json:"visa,omitempty" example:"A12345678"`
	WhoisRegistrant           *string  `json:"whois-registrant,omitempty" example:"Example Organization Inc."`
	WhoisRegistrar            *string  `json:"whois-registrar,omitempty" example:"GoDaddy.com, LLC"`
	WindowsScheduledTask      *string  `json:"windows-scheduled-task,omitempty" example:"\\Microsoft\\Windows\\Defrag\\ScheduledDefrag"`
	WindowsServiceDisplayName *string  `json:"windows-service-displayname,omitempty" example:"Windows Update"`
	WindowsServiceName        *string  `json:"windows-service-name,omitempty" example:"wuauserv"`
	Xmr                       *string  `json:"xmr,omitempty" example:"44AFFq5kSiGBoZ4NMDwYtN18obc8AemS33DBLWs3H7otXft3XjrpDtQGv7SqSsaBYBb98uNbr2VBBEt7f2wfn3RVGQBEP3A"`
	ZipCode                   *string  `json:"zip-code,omitempty" example:"10001"`
}

// GetAttribute returns the value of the attribute with the specified JSON tag name.
// It returns the attribute value and a boolean indicating whether the attribute was found.
// If the attribute isn't found, it returns nil and false.
func (d *Attributes) GetAttribute(tagName string) (interface{}, bool) {
	val := reflect.ValueOf(d).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("json")

		tagParts := strings.Split(tag, ",")
		if tagParts[0] == tagName {
			fieldValue := val.Field(i)

			if fieldValue.IsNil() {
				return nil, true
			}

			return fieldValue.Elem().Interface(), true
		}
	}

	return nil, false
}

// ToMap returns all existing attributes and values of the Definition as a map[string]interface{}
func (d *Attributes) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	val := reflect.ValueOf(d).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		tag := field.Tag.Get("json")
		tagParts := strings.Split(tag, ",")
		jsonName := tagParts[0]

		if fieldValue.IsZero() {
			continue
		}

		result[jsonName] = fieldValue.Interface()
	}

	return result
}

// SetAttribute sets the value of the attribute with the specified JSON tag name.
// It returns a boolean indicating whether the attribute was found and set successfully.
// If the attribute isn't found, it returns false.
func (d *Attributes) SetAttribute(tagName string, value interface{}) bool {
	val := reflect.ValueOf(d).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("json")

		tagParts := strings.Split(tag, ",")
		if tagParts[0] == tagName {
			fieldValue := val.Field(i)

			if field.Type.Kind() == reflect.Ptr {
				elemType := field.Type.Elem()

				newValue := reflect.New(elemType)

				switch elemType.Kind() {
				case reflect.String:
					if strVal, ok := value.(string); ok {
						newValue.Elem().SetString(strVal)
					} else if value == nil {
						fieldValue.Set(reflect.Zero(field.Type))
						return true
					} else {
						newValue.Elem().SetString(fmt.Sprintf("%v", value))
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if intVal, ok := value.(int); ok {
						newValue.Elem().SetInt(int64(intVal))
					} else if int64Val, ok := value.(int64); ok {
						newValue.Elem().SetInt(int64Val)
					} else if floatVal, ok := value.(float64); ok {
						newValue.Elem().SetInt(int64(floatVal))
					} else if value == nil {
						fieldValue.Set(reflect.Zero(field.Type))
						return true
					} else {
						if strVal, ok := value.(string); ok {
							if intVal, err := strconv.ParseInt(strVal, 10, 64); err == nil {
								newValue.Elem().SetInt(intVal)
							} else {
								return false
							}
						} else {
							return false
						}
					}
				case reflect.Float32, reflect.Float64:
					if floatVal, ok := value.(float64); ok {
						newValue.Elem().SetFloat(floatVal)
					} else if intVal, ok := value.(int); ok {
						newValue.Elem().SetFloat(float64(intVal))
					} else if int64Val, ok := value.(int64); ok {
						newValue.Elem().SetFloat(float64(int64Val))
					} else if value == nil {
						fieldValue.Set(reflect.Zero(field.Type))
						return true
					} else {
						if strVal, ok := value.(string); ok {
							if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
								newValue.Elem().SetFloat(floatVal)
							} else {
								return false
							}
						} else {
							return false
						}
					}
				default:
					return false
				}

				fieldValue.Set(newValue)
			} else {
				return false
			}

			return true
		}
	}

	return false
}
