# HL7 v2 MLLP to Aidbox proxy

This tiny utility is intended to capture HL7 v2 traffic and submit
every received message to the Aidbox FHIR Server where message parsing
and mapping will happen.

Most likely you'll want to deploy this utility to some host inside
your organization's infrastructure to avoid setting up VPN or any
other tunneling to external network. `hl7proxy` is self-sufficient
binary which means it doesn't have any dependencies (DLLs, packages,
etc). All you need is to pick a binary for your OS/architecture, drop
to target host and then run it.

Precompiled binaries are available on
[Releases](https://github.com/HealthSamurai/hl7proxy/releases)
page.
## Configuring Aidbox

Before running `hl7proxy` utility you need to configure your Aidbox
instance with proper `Hl7v2Config` resource. Follow [HL7V2-IN
documentation](https://docs.aidbox.app/advanced/hl7-v2-integration)
for detailed instructions.

Also you'll need to make a Client resource to authorize
`hl7proxy` HTTP requests. Follow [Basic Auth
tutorial](https://docs.aidbox.app/auth-betta/basic-auth) to get a
value for `Authorization` header and then pass it to `hl7proxy`
utility with `-header` command-line flag.

## Usage

`hl7proxy` is a command-line utility, so you need to start Terminal or
Console application for your operating system. Also on Linux/OSX
you'll need to make binary executable after downloading it:

``` shell
chmod +x ~/Downloads/hl7proxy-linux-amd64
```

(don't forget to alter the filename if you use different
OS/architecture)

`cd` into the directory where `hl7proxy` binary is located and invoke
it:

``` shell
./hl7proxy -port 4242 -config default -url https://your-box.edge.aidbox.app/ -header "Authorization: Basic xxxxxxxxxxxxx"
```

Make sure you changed `-header`, `-url` and `-config` values to match
your Aidbox instance configuration.

If you see a message `Listening to :4242` then `hl7proxy` is running
and ready to accept connections. You can change port number with
`-port` flag.

You could emulate message delivery with MLLP using `netcat`:
``` shell
## replace test.hl7 with a path to file with one hl7 message
echo -n -e "\x0b$(cat test.hl7)\x1c\x0d\x0b" | nc localhost 4242 | tr -d '\013\034' | tr '\015' '\n'
```

or use provided script to send multiple messages:

``` shell
./try/send.sh ./try/msg1.hl7 ./try/msg2.hl7
## MSH|^~\&|||AccMgr|1|2023721115350||ACK||P|2.3
## MSA|AA|599102||||

## MSH|^~\&|HOMERTON_TIE|HOMERTON|P0241|HOMERTON|2023721115350||ACK|||2.3
## MSA|AA|Q111111119T4083493511111111||||
```

expected output of the `hl7proxy`:

``` shell
2023/07/21 11:53:47 INFO: Listening to :4242
2023/07/21 11:53:50 INFO: New connection from [::1]:56259
2023/07/21 11:53:50 INFO: New message received. Length: 1500 bytes
2023/07/21 11:53:50 INFO: Message identity: type: ADT^A01, ctlid: 599102, from: 1, to:
2023/07/21 11:53:50 INFO: Message delivered to Aidbox: 201 Created (25ms)
2023/07/21 11:53:50 INFO: Sending ACK
2023/07/21 11:53:50 INFO: New message received. Length: 1999 bytes
2023/07/21 11:53:50 INFO: Message identity: type: ADT^A31, ctlid: Q111111119T4083493511111111, from: HOMERTON, to: HOMERTON
2023/07/21 11:53:51 INFO: Message delivered to Aidbox: 201 Created (18ms)
2023/07/21 11:53:51 INFO: Sending ACK
2023/07/21 11:53:51 INFO: Client [::1]:56259 disconnected
```


## Running as a Windows service

To be written. Short answer: use [NSSM](http://nssm.cc/) to register
`hl7proxy` as a Service.

## Support

Feel free to join ["Aidbox Users" Telegram group](https://t.me/aidbox)
to ask any questions regarding `hl7proxy` or Aidbox. Also you can
create an issue in this repo to report bug, misbehaviour or ask for a
feature request.

## License

Copyright © 2019 [Health Samurai](https://health-samurai.io/) team.

hl7proxy is released under the terms of the MIT License.
