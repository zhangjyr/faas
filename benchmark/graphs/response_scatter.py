import sys
import plotly
import plotly.graph_objs as go

from parsers.response import Response

def scatter(series):
    trace = go.Scatter(
        x = series[0],
        y = series[1],
        mode = "markers"
    )
    data = [trace]
    plotly.offline.plot(data, auto_open=True)

if __name__ == "__main__":
    file = None
    if len(sys.argv) > 1:
        file = sys.argv[1]

    response = Response.parse(file).filter(lambda record: record[Response.FIELD_STATUSCODE] == 200)
    scatter([response.series(Response.FIELD_TIME), response.series(Response.FIELD_RESPONSE)])
