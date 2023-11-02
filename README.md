# What does this solve?
I wanted one button to:

    - turn on my projector
    - switch projector input to HDMI2
    - open my projector screen
    - switch output on my PC to said projector while disabling my 3 monitors

This software does the last thing on that list.

# Usage
I assume you already have [autorandr](https://github.com/phillipberndt/autorandr) installed and saved at least one
configuration there (`autorandr --save NAME`)

You also need to have MQTT server running.

All configuration is done using environment variables.
```bash
go build
export MY_NAME=lukasz-pc
export MQTT_URL=tcp://homeassistant.host.name:1883
export MQTT_USERNAME=automation
export MQTT_PASSWORD=********
./autorandr2mqtt
```


Important: `MY_NAME` contains name that will appear in homeassistant and as unique_id. So you probably should keep it
alphanumeric :)

# TODO

This is list of what I think is missing. However, currently all functions that I need are implemented, so I will not
be adding these features. Feel free to open pull requests.

 - Mutual TLS authentication
 - refreshing of autorandr profiles during runtime, not only at startup
 - configurable homeassistant mqtt discovery prefix
 - throttling of incoming messages