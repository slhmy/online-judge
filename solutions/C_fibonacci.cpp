#include <iostream>
using namespace std;

int main() {
    int n;
    cin >> n;
    if (n <= 1) {
        cout << n << endl;
        return 0;
    }
    long long a = 0, b = 1;
    for (int i = 2; i <= n; i++) {
        long long c = a + b;
        a = b;
        b = c;
    }
    cout << b << endl;
    return 0;
}
