import requests
import time
import os
import sys
from threading import Thread
from time import sleep
from requests.adapters import HTTPAdapter

def newRequest(method, url, **kwargs):
    req = requests.Request(method, url, **kwargs)
    return req.prepare()

def send(req):
    with requests.Session() as session:
        session.mount('http://', HTTPAdapter(max_retries=0))
        return session.send(req, timeout=1)

reused = requests.Session()
reused.mount('http://', HTTPAdapter(max_retries=0))
def request(count, req, handler = None, reuse = False, entries = None):
    if entries == None:
        entries = []

    for j in range(count):
        start = time.time()
        status_code = 500
        try:
            if reuse:
                r = reused.send(req, timeout=1)
            else:
                r = send(req)
            status_code = r.status_code
        except KeyboardInterrupt:
            raise
        except:
            pass
        end = time.time()
        elapse = 0

        fields = [start, status_code, end - start]
        if handler != None:
            fields.extend(handler(r))

        entries.append(fields)

    return entries


class RequestThread(Thread):
    """docstring for MyThread"""

    def __init__(self, count, req, entries, handler = None, reuse = False):
        super(RequestThread, self).__init__()
        self.count = count
        self.req = req
        self.handler = handler
        self.reuse = reuse
        self.entries = entries


    def run(self):
        request(self.count, self.req, entries = self.entries, handler = self.handler, reuse = self.reuse)


def threadize(total, req, num = 1, handler = None, reuse = False):
    per = int(round(total / num))
    total2 = per * num
    rest = per - (per * num - total2)
    entries = []

    threads = []
    for i in range(0, num - 1):
        threads.append(RequestThread(per, req, entries, handler = handler, reuse = reuse))
    threads.append(RequestThread(rest, req, entries, handler = handler, reuse = reuse))

    for thread in threads:
        thread.start()
    for thread in threads:
        thread.join()

    return entries
