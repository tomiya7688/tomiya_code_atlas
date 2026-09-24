#include <future>
#include "support.hpp"

#define TOMIYA_TOUCH(value) helper(value)

class BaseWorker {
};

template <typename T>
class Worker : public BaseWorker {
public:
    T run(T item) {
        TOMIYA_TOUCH(item);
        helper(item);
        return item;
    }

    T helper(T item) {
        return item;
    }

    T helper(T item, int count) {
        return count > 0 ? item : helper(item);
    }

    std::future<T> async_probe(T item) {
        return std::async(std::launch::async, [item]() { return item; });
    }

    T nested_probe(T item) {
        auto nested = [](T value) { return value; };
        return nested(item);
    }
};
