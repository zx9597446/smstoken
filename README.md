# smstoken
an sms token service that generate/validation sms token. (using twilio to send sms for now)

# requirements
1. redis to save generated tokens.
2. twilio account

# install

	go get -u github.com/zx9597446/smstoken

# configure
simple run `smstoken` will generate `config.conf` in current direcotry, fill required fileds then run `smstoken` again. 
```
{
  "SecretKey": "abcd",
  "TwilioId": "abcd",
  "TwilioKey": "abcd",
  "TokenKeepAliveInSeconds": 3600,
  "PreventSendInSeconds": 60,
  "TokenLength": 6 
}
```

# command line arguments

```
Usage of smstoken:
  -addr=":8080": address to listen
  -c="config.conf": config file in json
  -redis="127.0.0.1:6379": redis address
```

# APIs
1. request send sms token.
```
	Method: POST
	URI:	/send
	Post Body(in JSON): {"From": "who sent sms", "To": "send sms to whom", "Text": "text before token"}
	Returns: HTTP Code 200 or 400 or 500
```

2. request verify sms token.
```
	Method: GET
	URI:	/validation/:phone/:token (NOTE: :phone and :token is actual phone number and token number here, eg: /validation/2343434/334343)
	Returns: HTTP Code 200: { "result": true/false } or 400 or 500
```


# API signature
formation of signature in request header is:

	timestamp,signature

to generate signature:

	signature = md5(timestamp + secret)

then set request header `X-Sms-Signature` 
