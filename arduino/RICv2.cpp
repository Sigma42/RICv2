//
// Created by sigma42 on 01.05.24.
//

#include "RICv2.h"

RobotikInterConnect::RobotikInterConnect(unsigned char address) {
    Serial.setTimeout(1000); //Set Timeout to 1 Seconds
    Serial.begin(57600); //9600 - Safe Speed //115200 - Almost Fastest

    this->address = address;

    //Assumes USB Connection already exists and server is running
    RIC_PCK p = {.version = this->version, .src = address, .dst = 0, .flags = 1 | 2};
    send(p);
}

bool RobotikInterConnect::can_send() {
    return Serial.availableForWrite() < sizeof(RIC_PCK);
}

bool RobotikInterConnect::can_recv() {
    return Serial.available() < sizeof(RIC_PCK);
}

//Have to set target and data
void RobotikInterConnect::send(RIC_PCK& p) {
    p.version = this->version;
    p.src = this->address;
    p.flags = 0;

    while (!can_send()) delay(2);

    Serial.write((uint8_t*)&p,sizeof(RIC_PCK));

    for (int i=0; i < 20; i++) {
        p.data[i] = 0;
    }
}

void RobotikInterConnect::recv(RIC_PCK& p) {
    while (!can_recv()) delay(2);

    Serial.readBytes((uint8_t*)&p,sizeof(RIC_PCK));
}