def ft_count_harvest_recursive():
    def count(day, days):
        if day > days:
            print("Harvest time!")
            return
        print("Day", day)
        count(day + 1, days)

    count(1, int(input("Days until harvest: ")))
