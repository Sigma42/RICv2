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

    RIC_PCK(): version(1), src(0), dst(0), flags(0) {
        for (int i = 0; i < 20; i++) {
            data[i] = 0;
        }
    }

    RIC_PCK(unsigned char v, unsigned char s, unsigned char d, unsigned char f): version(v), src(s), dst(d), flags(f) {
        for (int i = 0; i < 20; i++) {
            data[i] = 0;
        }
    }
} RIC_PCK;

class RobotikInterConnect {
private:

public:
    unsigned char version = 1;
    unsigned char address;
    
    char inStringBuffer[21];

    RobotikInterConnect(unsigned char address);
    bool can_recv();
    bool can_send();
    void send(RIC_PCK &p);
    void recv(RIC_PCK &p);

    void send(unsigned char target, String msg);
    String recv();
};


#endif // RIC_HPP