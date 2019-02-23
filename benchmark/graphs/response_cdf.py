import sys
import plotly
import plotly.graph_objs as go
import numpy as np

from parsers.response import Response

def prepare(trace, name):
    trace.sort()
    cumsum = np.cumsum(trace)

    data = go.Scatter(
        x = [trace[i] for i in range(len(cumsum))],
        y = [(i + 0.5) / len(cumsum) for i in range(len(cumsum))],
        name = name
    )

    return data

def cdf(traces):
    data = map(lambda trace: prepare(trace.series(Response.FIELD_RESPONSE), trace.name), traces)

    layout = go.Layout(
        xaxis=dict(
            # type='log',
            autorange=True
        )
    )

    fig = go.Figure(data = data, layout = layout)
    plotly.plotly.iplot(fig, auto_open = True)

if __name__ == "__main__":
    files = []
    for i in range(len(sys.argv) - 1):
        files.append(sys.argv[i + 1])

    cdf(map(
        lambda file: Response.parse(file).filter(
            lambda record: record[Response.FIELD_STATUSCODE] == 200
        ),
        files
    ))
