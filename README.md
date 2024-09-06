<br /><br />

<p align="center"><img align="center" src="https://customer-os.imgix.net/companies/logos/mailsherpa_logo.png" height="200" alt="mailsherpa" /></p>
<h1 align="center">MailSherpa</h1>
<h4 align="center">A CLI for verifying email address deliverability over SMTP without sending an email.</h4>

<br /><br /><br />

## 👉 Live Demo: https://customeros.ai

This is open-source, but we also offer a hosted API that's simple to use. If you are interested, find out more at [CustomerOS](https://docs.customeros.ai/api-reference/verify/verify-an-email-address). If you have any questions, you can contact me at matt@customeros.ai.

<br />

## Installation 

If you want to use our install script, you can run the following command:

```
curl -sSL https://mailsherpa.sh/install.py | python3
```
otherwise, follow the diretions below:

Download the appropriate CLI tarball for your OS:

```
wget mailsherpa.sh/mailsherpa-linux-arm64.tar.gz
wget mailsherpa.sh/mailsherpa-linux-amd64.tar.gz
wget mailsherpa.sh/mailsherpa-macos.tar.gz
```

Extract the binary:

```
tar -xzf filename.tar.gz
```

4. Test to make sure everything is working

```
./mailsherpa version
```

## Set env variables

Set the `MAIL_SERVER_DOMAIN` environment variable.  See the `Mail Server setup guide` section below for more details:

```
export MAIL_SERVER_DOMAIN=example.com
```


## Mail Server setup guide

You might be asking why you need to setup a mail server...
