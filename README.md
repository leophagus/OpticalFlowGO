# Optical Flow in GO

Lucas Kanade Optical Flow implemented in GO. This is non-pyramidal, non-iterative version of LK. Uses CSP approach. Each process (Goroutine) implements portions of the algorithm and passes the partial results to the next  process(es). Such a design style is very natural in RTL based hardware design. Learned GO just to play with CSP and Goroutines. This is my first Go project.

The image derivatives within the window are computed incrementally. Compute and add right-column. Compute and subtract left-column. Additional optimizations are possible to make it O(1) with respect to window-size.
