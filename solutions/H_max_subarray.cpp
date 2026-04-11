#include <iostream>
#include <climits>
using namespace std;

int main() {
    int n;
    cin >> n;
    long long maxSum = LLONG_MIN, curSum = 0;
    for (int i = 0; i < n; i++) {
        int x;
        cin >> x;
        curSum += x;
        if (curSum > maxSum) maxSum = curSum;
        if (curSum < 0) curSum = 0;
    }
    cout << maxSum << endl;
    return 0;
}
