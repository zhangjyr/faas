import sys
import os
import plotly
import plotly.graph_objs as go

from parsers.response import Response

def box(responses):
    data = map(lambda response: go.Box(
        y = response.series(Response.FIELD_RESPONSE),
        x = map(lambda time: int(time), response.series(Response.FIELD_TIME)),
        boxmean = True,
        name = response.name,
    ), responses)

    layout = go.Layout(
        yaxis=dict(
            type='log',
            autorange=True
        )
    )

    fig = go.Figure(data = data, layout = layout)
    plotly.offline.plot(fig, auto_open = True)

if __name__ == "__main__":
    files = []
    for i in range(len(sys.argv) - 1):
        files.append(sys.argv[i + 1])

    box(map(
        lambda file: Response.parse(file, os.path.splitext(os.path.basename(file))[0].replace("{0}", "")).filter(
            lambda record: record[Response.FIELD_STATUSCODE] == 200 and record[Response.FIELD_TIME] < 30
        ),
        files
    ))
