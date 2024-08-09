package mailserver

import "testing"

func TestParseSmtpResponse(t *testing.T) {
	testCases := []struct {
		input         string
		expectedCode  string
		expectedError string
		expectedDesc  string
	}{
		{
			input:         "451 Internal resource temporarily unavailable - https://community.mimecast.com/docs/DOC-1369#451 [V1X4ES abPxKDmn8czHBhOQ.uk246]",
			expectedCode:  "451",
			expectedError: "",
			expectedDesc:  "Internal resource temporarily unavailable - https://community.mimecast.com/docs/DOC-1369#451 [V1X4ES abPxKDmn8czHBhOQ.uk246]",
		},
		{
			input:         "550 5.4.1 Recipient address rejected: Access denied. [TY1PEPF0000BAD7.JPNP286.PROD.OUTLOOK.COM 2024-08-06 T19:13:55.906Z 08DCB6409251B38F]",
			expectedCode:  "550",
			expectedError: "5.4.1",
			expectedDesc:  "Recipient address rejected: Access denied. [TY1PEPF0000BAD7.JPNP286.PROD.OUTLOOK.COM 2024-08-06 T19:13:55.906Z 08DCB6409251B38F]",
		},
		{
			input:         "550 5.7.1 XGEMAIL_0011 Command rejected",
			expectedCode:  "550",
			expectedError: "5.7.1",
			expectedDesc:  "XGEMAIL_0011 Command rejected",
		},
		{
			input:         "554 Reject by behaviour spam at Rcpt State(Connection IP address:65.108.247.80)ANTISPAM_BAT[0120131 1R156a, maildocker-behaviorspam033018096094]: spf check failedCONTINUE",
			expectedCode:  "554",
			expectedError: "",
			expectedDesc:  "Reject by behaviour spam at Rcpt State(Connection IP address:65.108.247.80)ANTISPAM_BAT[0120131 1R156a, maildocker-behaviorspam033018096094]: spf check failedCONTINUE",
		},
		{
			input:         "550 5.1.1 <paizal.ke@almana.com>: Email address could not be found, or was misspelled (G8)",
			expectedCode:  "550",
			expectedError: "5.1.1",
			expectedDesc:  "<paizal.ke@almana.com>: Email address could not be found, or was misspelled (G8)",
		},
		{
			input:         "550 Invalid recipient <ebun.adebonojo@bbc.com> (#5.1.1)",
			expectedCode:  "550",
			expectedError: "5.1.1",
			expectedDesc:  "Invalid recipient <ebun.adebonojo@bbc.com> (#5.1.1)",
		},
		{
			input:         "550-5.2.1 The email account that you tried to reach is inactive. For more",
			expectedCode:  "550",
			expectedError: "5.2.1",
			expectedDesc:  "The email account that you tried to reach is inactive. For more",
		},
		{
			input:         "550 #5.1.0 Address rejected.",
			expectedCode:  "550",
			expectedError: "5.1.0",
			expectedDesc:  "Address rejected.",
		},
		{
			input:         "550-#-5.1.0-Address rejected.",
			expectedCode:  "550",
			expectedError: "5.1.0",
			expectedDesc:  "Address rejected.",
		},
		{
			input:         "550 5.7.1 Service unavailable, Client host [92.239.49.239] blocked using Spamhaus. To request removal from this list see https://www.spamhaus.org/query/ip/92.239.49.239 AS(1450) [LO1PEPF000022FC.GBRP265.PROD.OUTLOOK.COM 2024-08-08T21:41:59.031Z 08DCB37616E26EE1]",
			expectedCode:  "550",
			expectedError: "5.7.1",
			expectedDesc:  "Service unavailable, Client host [92.239.49.239] blocked using Spamhaus. To request removal from this list see https://www.spamhaus.org/query/ip/92.239.49.239 AS(1450) [LO1PEPF000022FC.GBRP265.PROD.OUTLOOK.COM 2024-08-08T21:41:59.031Z 08DCB37616E26EE1]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			code, err, desc := ParseSmtpResponse(tc.input)
			if code != tc.expectedCode {
				t.Errorf("expected status code %s, got %s", tc.expectedCode, code)
			}
			if err != tc.expectedError {
				t.Errorf("expected error code %s, got %s", tc.expectedError, err)
			}
			if desc != tc.expectedDesc {
				t.Errorf("expected description %s, got %s", tc.expectedDesc, desc)
			}
		})
	}
}
