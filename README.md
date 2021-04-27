Containerized Golang application to be run on relay laptops that are in physical proximity of Bluetooth Low Energy (BLE) devices.

This application was deployed to interface with multiple [DFRobot Beetle BLEs](https://www.dfrobot.com/product-1259.html), and stream data to an upstream neural network, as part of a dance moves detector.

- Reliable custom protocol w/ automatic reconnection
- Windowed (of configurable size) data streams
- Built-in data collection
- Portable

Tested with 4 active connections * 120 packets (containing 20 payload bytes each) / sec.

- informally observed to be able to support much higher avg load (e.g. 6 beetles at once)
- YMMV due to hardware considerations & performance, e.g. bluetooth adapter used


![](https://user-images.githubusercontent.com/40201586/106851817-e2040d80-66f1-11eb-819b-36ccb35d8eb6.png)
