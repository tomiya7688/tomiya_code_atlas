package fixture;

import java.util.concurrent.CompletableFuture;
import java.util.function.Function;
import static java.util.Objects.requireNonNull;
import support.Service;

interface WorkContract<T> {
    T run(T item);
}

enum State {
    READY,
    RUNNING;
}

@interface Marker {
    String value() default "fixture";
}

record Pair<T>(@Marker("left") T left, T right) {
}

record Checked(int value) {
    Checked {
        validate(value);
    }

    private static void validate(int value) {
    }
}

class BaseWorker {
}

class Worker<T> extends BaseWorker implements WorkContract<T> {
    private final T current;

    /** Creates a worker. */
    Worker(T current) {
        this.current = requireNonNull(current);

        class DeferredCall {
            void later() {
                helper(current);
            }
        }
    }

    static class Nested {
        int count;
    }

    @Override
    public T run(T item) {
        helper(item);
        helper(item);
        return item;
    }

    T helper(T item) {
        return item;
    }

    T helper(T item, int count) {
        return item;
    }

    <N extends Number> Worker(N number, T current) {
        this(current);
    }

    native int read();

    void log(String... values) {
    }

    CompletableFuture<T> async_probe(T item) {
        return CompletableFuture.completedFuture(item);
    }

    T nested_probe(T item) {
        Function<T, T> nested = value -> value;
        return nested.apply(item);
    }
}

