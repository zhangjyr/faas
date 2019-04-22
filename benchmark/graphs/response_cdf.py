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
    plotly.offline.plot(fig, auto_open = True)

if __name__ == "__main__":
    files = []
    skip = False
    for i in range(1, len(sys.argv)):
        if skip:
            skip = False
            continue

        file = sys.argv[i]
        if file.replace("{0}", "") != file and len(sys.argv) > i + 1:
            files = files + map(
                lambda postfix: file.format("_" + postfix), sys.argv[i + 1].split(",")
            )
            skip = True
        else:
            files.append(file)

    cdf(map(
        lambda file: Response.parse(file).filter(
            lambda record: record[Response.FIELD_STATUSCODE] == 200
        ),
        files
    ))
