import sys
import os
import plotly
import plotly.graph_objs as go
import numpy as np

from parsers.response import Response

def prepare(trace, name):
    data = go.Histogram(
        x=trace,
        nbinsx=1000,
        name=name
    )
    return data

def histogram(traces):
    data = map(lambda trace: prepare(trace.series(Response.FIELD_RESPONSE), trace.name), traces)

    layout = go.Layout(
    )

    fig = go.Figure(data = data, layout = layout)
    plotly.plotly.iplot(fig, auto_open = True)

if __name__ == "__main__":
    base = os.path.abspath('.')

    files = []
    for i in range(len(sys.argv) - 1):
        files.append(sys.argv[i + 1])

    histogram(map(
        lambda file: Response.parse(file).filter(
            lambda record: record[Response.FIELD_STATUSCODE] == 200
        ),
        files
    ))
