def package_dependency_resolver(packages: dict[str, list[str]]) -> list[str]:
    order = []
    installed = set()
    remaining = set(packages)
    while remaining:
        ready = sorted(
            name for name in remaining
            if all(dep in installed or dep not in packages
                   for dep in packages[name])
        )
        if not ready:
            return []
        order.extend(ready)
        installed.update(ready)
        remaining.difference_update(ready)
    return order
