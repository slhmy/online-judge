#include <iostream>
#include <vector>
#include <unordered_map>
using namespace std;

int main() {
    int n;
    cin >> n;
    vector<int> a(n);
    for (int i = 0; i < n; i++) cin >> a[i];
    int target;
    cin >> target;

    unordered_map<int, int> mp;
    for (int i = 0; i < n; i++) {
        int complement = target - a[i];
        if (mp.count(complement)) {
            cout << mp[complement] << " " << i << endl;
            return 0;
        }
        mp[a[i]] = i;
    }
    return 0;
}
