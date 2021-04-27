# comms int

Containerized Golang application to be run on relay laptops that are in physical proximity of Bluetooth Low Energy (BLE) devices. This was used to interface with multiple DFRobot Beetle BLEs, and stream data to upstream sybsystems as part of a dance moves detector.

Features:
- Reliable custom protocol w/ automatic reconnection
- Tested to work with 3 beetles * ~120 packets, 20 bytes each / sec, informally observed to be able to support more load (e.g. 6 beetles at once)
- Windowed (size is configurable) stream of data to upstream applications (e.g. neural networks)
- Built-in data collection

Note: if you experience undocumented issues with serial communication, <b>rule out power supply issues</b>

![](https://user-images.githubusercontent.com/40201586/106851817-e2040d80-66f1-11eb-819b-36ccb35d8eb6.png)
