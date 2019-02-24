import sys
import os
import plotly
import plotly.graph_objs as go
import numpy as np

from parsers.response import Response

def prepare_latency(res, pencentile):
    trace = res.series(Response.FIELD_RESPONSE)
    trace.sort()
    cumsum = np.cumsum(trace)

    return trace[int(round(pencentile * len(cumsum) - 0.5))]

def prepare_throuput(res):
    trace = res.series(Response.FIELD_TIME)
    trace.sort()

    return round(len(trace) / trace[len(trace) - 1])

def prepare(trace, pencentile):
    trace = go.Scatter(
        x = map(lambda response: prepare_throuput(response), trace),
        y = map(lambda response: prepare_latency(response, pencentile), trace),
        mode = "lines+markers",
        name = trace[0].name
    )
    return trace

def scatter(traces):
    percentile = 0.5
    data = map(lambda trace: prepare(trace, percentile), traces)

    layout = go.Layout(
        title = str(int(percentile * 100)) + "% Percentile"
    )

    fig = go.Figure(data = data, layout = layout)
    plotly.plotly.iplot(fig, auto_open=True)

# python graphs/response_scatter_throughput_latency.py file_pattern postfixes[ file_pattern_2 postfixes_2]
# Example python graphs/response_scatter_throughput_latency.py data/10_hello{0}.csv 1,2,4,8,16,32
if __name__ == "__main__":
    groups = []
    for i in range(1, len(sys.argv), 2):
        file = sys.argv[i]
        name = os.path.splitext(os.path.basename(file))[0].replace("{0}", "")
        groups.append([ name, map(
            lambda postfix: file.format("_" + postfix), sys.argv[i + 1].split(",")
        )])

    scatter(map(
        lambda group: map(
            lambda file: Response.parse(file, group[0]).filter(
                lambda record: record[Response.FIELD_STATUSCODE] == 200
            ),
            group[1]
        ),
        groups
    ))
