import math
import sys
from scipy import stats

def run(target, confidence, problem, args = None, n = 1, creator = False):
    history = None

    q = 1 - (1 - confidence) / 2
    z = stats.norm.ppf(q)

    accuracy = 100
    cnt = 0
    sum = 0
    sum2 = 0
    if creator:
        problem = problem()
    while accuracy > target:
        ret = problem(n, args = args)
        cnt += 1
        sum += ret
        sum2 += pow(ret, 2)

        if cnt > 1:
            mean = sum / cnt
            var = (cnt * sum2 - pow(sum, 2)) / (cnt * (cnt - 1))
            ppf = z
            if cnt < 30:
                ppf = stats.t.ppf(q, cnt - 1)
            accuracy = ppf * math.sqrt(var / cnt) / mean * 100

            print("iteration:{0}\tmean:{1}\tprecision:{2}".format(cnt, mean, accuracy))
            sys.stdout.flush()

    print("Total iterations: {0}, precision: {1}".format(cnt, round(accuracy, 3)))
    return sum / cnt
