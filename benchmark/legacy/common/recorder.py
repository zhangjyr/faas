from threading import Thread, Event, Lock

class Recorder(Thread):
    """docstring for MyThread"""

    def __init__(self, path):
        super(Recorder, self).__init__()
        self.path = path
        self.entries = None
        self.shouldClose = False
        self.filename = 'response.csv'
        self.file = None
        self.newData = Event()
        self.mu = Lock()

    def run(self):
        while True:
            self.newData.wait()

            self.mu.acquire()
            shouldClose = self.shouldClose
            entries = self.entries
            self.entries = None
            self.newData.clear()
            self.mu.release()

            if self.file == None:
                self.file = open(self.path + self.filename, 'w')

            if entries != None:
                for i in entries:
                    self.file.write(",".join(map(lambda field: str(field), i)))
                    self.file.write('\n')

            if shouldClose:
                self.file.close()
                break

    def open(self, filename = None):
        if self.file == None:
            if filename != None:
                self.filename = filename
            self.start()

    def save(self, entries, filename = None):
        if filename != None:
            self.open(filename)

        self.mu.acquire()
        if self.entries == None:
            self.entries = entries
        else:
            self.entries.extend(entries)
        self.newData.set()
        self.mu.release()

    def close(self):
        self.mu.acquire()
        self.shouldClose = True
        self.newData.set()
        self.mu.release()

        self.join()
