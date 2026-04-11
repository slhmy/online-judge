#include <iostream>
using namespace std;

int main() {
    long long n;
    cin >> n;
    for (long long i = 2; i * i <= n; i++) {
        int cnt = 0;
        while (n % i == 0) {
            cnt++;
            n /= i;
        }
        if (cnt > 0) cout << i << " " << cnt << endl;
    }
    if (n > 1) cout << n << " 1" << endl;
    return 0;
}
