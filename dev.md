# On release

```
git tag v0.x.x
git push
git push --tags (or git push origin v0.x.x)
GOPROXY=proxy.golang.org go list -m github.com/RottenNinja-Go/framework@v0.x.x
```