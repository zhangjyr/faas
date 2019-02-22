import sys
import plotly
import plotly.graph_objs as go

from parsers.response import Response

def box(responses):
    data = map(lambda response: go.Box(
        y = response.series(Response.FIELD_RESPONSE),
        x = map(lambda time: int(time) / 10 * 10, response.series(Response.FIELD_TIME)),
        boxmean = True,
    ), responses)

    layout = go.Layout(
    )

    fig = go.Figure(data = data, layout = layout)
    plotly.offline.plot(fig, auto_open = True)

if __name__ == "__main__":
    files = []
    for i in range(len(sys.argv) - 1):
        files.append(sys.argv[i + 1])

    box(map(
        lambda file: Response.parse(file).filter(
            lambda record: record[Response.FIELD_STATUSCODE] == 200
        ),
        files
    ))
