import requests
import time
import os
import sys
from threading import Thread
from time import sleep

def newRequest(method, url, **kwargs):
    req = requests.Request(method, url, **kwargs)
    return req.prepare()

def send(req):
    with requests.Session() as session:
        return session.send(req)

reused = requests.Session()
def request(count, req, handler = None, reuse = False):
    entries = []
    for j in range(count):
        start = time.time()
        if reuse:
            r = reused.send(req)
        else:
            r = send(req)
        end = time.time()
        elapse = 0

        fields = [start, r.status_code, end - start]
        if handler != None:
            fields.extend(handler(r))

        entries.append(fields)
        return entries


class RequestThread(Thread):
    """docstring for MyThread"""

    def __init__(self, count):
        super(MyThread, self).__init__()
        self.count = count

    def run(self):
        request(self.count)


def threadize(total, num = 1):
    per = round(total / num)
    total2 = per * num
    rest = per - (per * num - total2)

    threads = []
    for i in range(0, num - 1):
        threads.append(RequestThread(per))
    threads.append(RequestThread(rest))

    for thread in threads:
        thread.start()
    for thread in threads:
        thread.join()
