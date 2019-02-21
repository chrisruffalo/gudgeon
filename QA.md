# Questions and Answers

## Q: Why is Gudgeon statically compiled?
This has been something I've debated with myself a lot and, in the end, the basic answer is that I wanted a portable binary that could run by itself almost anywhere you put it. It does mean that if there are security issues with the underlying libraries it may take a while to get around to pushing the fixes out but that risk is minor. If necessary the needed to statically compile Gudgeon can be changed or both static and non-static binaries can be selectively produced.

## Q: Why are the resources baked into the binary?
Ease of distribution, really. The binary works by itself and that makes it easier to deal with. I think, in future versions, it may change either to use something other than Patternfly or to also be able to read from disk locations.

## Q: Why is the code organized this way?
I think my roots as a Java programmer are showing. Organizing packages this way is new to me at a large scale but I still want to maintain some isolation or at least separation of concerns. As packages start to tangle they get resolved down into a single package to manage the shared domain.