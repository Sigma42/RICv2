#ifndef RIC_HPP
#define RIC_HPP

#include "Arduino.h"

typedef unsigned char RIC_DATA;

typedef struct __attribute__ ((packed)) RIC_PCK {
    unsigned char version;
    unsigned char src;
    unsigned char dst;
    unsigned char flags;

    RIC_DATA data[20];
} RIC_PCK;

class RobotikInterConnect {
private:

public:
    unsigned char version = 1;
    unsigned char address;

    RobotikInterConnect(unsigned char address);
    bool can_send();
    bool can_recv();
    void send(RIC_PCK &p);
    void recv(RIC_PCK &p);
};

#endif // RIC_HPP