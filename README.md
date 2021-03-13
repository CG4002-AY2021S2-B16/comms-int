# comms int

Golang application to be run on relay laptops that are in physical proximity of BLE-compatible devices

- Up to 120 twenty-byte packets per second, per connected peripheral, over Bluetooth Low Energy v4.0
- Reliable protocol w/ automatic reconnection
- Windowed (size is configurable) stream of data to upstream applications (e.g. neural networks)

Note: if you experience undocumented issues with serial communication: <b>rule out power supply issues</b>

![](https://user-images.githubusercontent.com/40201586/106851817-e2040d80-66f1-11eb-819b-36ccb35d8eb6.png)
