#!/usr/bin/env python
import msgpack
from cocaine.asio import message


def output(data):
    return msgpack.packb(message.Message(*data).pack())


MESSAGES = ((message.RPC_HEARTBEAT, 0),
            (message.RPC_CHOKE, 1),
            (message.RPC_CHUNK, 1, "data"),
            (message.RPC_ERROR, 1, 99, "someerrormessage"))

print map(output, MESSAGES)
