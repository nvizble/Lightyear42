def py_graph_cycle_detector(graph: dict[int, list[int]]) -> bool:
    # 0 = unseen, 1 = on the current path, 2 = done; iterative DFS
    state = {}
    for root in graph:
        if state.get(root):
            continue
        state[root] = 1
        stack = [(root, iter(graph.get(root, [])))]
        while stack:
            node, it = stack[-1]
            for nxt in it:
                s = state.get(nxt, 0)
                if s == 1:
                    return True
                if s == 0:
                    state[nxt] = 1
                    stack.append((nxt, iter(graph.get(nxt, []))))
                    break
            else:
                state[node] = 2
                stack.pop()
    return False
