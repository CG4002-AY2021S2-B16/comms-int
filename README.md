This repository contains a set of Bluetooth Low Energy (BLE) applications that facilitate data delivery from wearable sensors to upstream systems. In particular, this setup was deployed to interface relay laptops with multiple [DFRobot Beetle BLEs](https://www.dfrobot.com/product-1259.html), and stream sensor data to an upstream neural network, as part of a larger dance moves trainer system. 

The code in this repository include a relevant selection of:
- C++/Arduino code to be run on sensors,
- Go code to run a BLE central application in physical proximity to sensors,
- Python code to transfer data to upstream components over the internet

The stack provides:
- Reliable custom protocol w/ automatic reconnection
- Windowed (of configurable size) data streams
- Built-in data collection
- Portability

This system has been tested with 4 active connections * 120 BLE 4.0 packets (containing 20 payload bytes each) / sec. However, it has been casually observed to be able to perform under much higher load.

![](https://user-images.githubusercontent.com/40201586/106851817-e2040d80-66f1-11eb-819b-36ccb35d8eb6.png)
