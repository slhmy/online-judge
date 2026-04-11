#include <iostream>
#include <string>
#include <cctype>
using namespace std;

int main() {
    string line;
    getline(cin, line);
    string clean;
    for (char c : line) {
        if (isalnum(c)) clean += tolower(c);
    }
    int l = 0, r = clean.size() - 1;
    bool ok = true;
    while (l < r) {
        if (clean[l] != clean[r]) { ok = false; break; }
        l++;
        r--;
    }
    cout << (ok ? "true" : "false") << endl;
    return 0;
}
