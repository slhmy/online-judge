#include <iostream>
#include <vector>
using namespace std;

int main() {
    int n;
    cin >> n;
    vector<int> a(n);
    for (int i = 0; i < n; i++) cin >> a[i];
    int target;
    cin >> target;

    int lo = 0, hi = n - 1, ans = -1;
    while (lo <= hi) {
        int mid = (lo + hi) / 2;
        if (a[mid] == target) { ans = mid; break; }
        else if (a[mid] < target) lo = mid + 1;
        else hi = mid - 1;
    }
    cout << ans << endl;
    return 0;
}
