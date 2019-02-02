
class Response:
    FIELD_TIME = 0
    FIELD_STATUSCODE = 1
    FIELD_RESPONSE = 2
    FIELD_ELAPSE = 3

    def __init__(self, records):
        self.records = records

    def series(self, field):
        return map(lambda record: record[field], self.records)

    def groupby(self, field, sort = False, mapper = None):
        groups = []
        hash = {}

        for i in range(len(self.records)):
            group = self.records[i][field]
            if mapper != None:
                group = mapper(group)

            if hash.get(group) == None:
                groups.append(group)
                hash[group] = []
            hash[group].append(self.records[i])

        if sort:
            groups.sort()

        ret = []
        for i in range(len(groups)):
            ret.append(Response(hash[groups[i]]))

        return ret

    def filter(self, func):
        return Response(filter(func, self.records))

    @staticmethod
    def parse(file):
        f = open(file,"r")

        start = 9999999999
        records = []
        for line in f:
            fields = line.split(",")
            fields[Response.FIELD_TIME] = float(fields[Response.FIELD_TIME])
            fields[Response.FIELD_STATUSCODE] = int(fields[Response.FIELD_STATUSCODE])
            fields[Response.FIELD_RESPONSE] = float(fields[Response.FIELD_RESPONSE])
            if fields[Response.FIELD_STATUSCODE] == 200:
                fields[Response.FIELD_ELAPSE] = float(fields[Response.FIELD_ELAPSE])
            else:
                fields[Response.FIELD_ELAPSE] = 0

            if fields[Response.FIELD_TIME] < start:
                start = fields[Response.FIELD_TIME]

            records.append(fields)

        for i in range(len(records)):
            records[i][Response.FIELD_TIME] = records[i][Response.FIELD_TIME] - start

        return Response(records)
