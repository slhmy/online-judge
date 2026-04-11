#include <iostream>
#include <string>
#include <vector>
using namespace std;

int main() {
    string text, pattern;
    cin >> text >> pattern;
    vector<int> positions;
    size_t pos = text.find(pattern, 0);
    while (pos != string::npos) {
        positions.push_back(pos);
        pos = text.find(pattern, pos + 1);
    }
    for (int i = 0; i < (int)positions.size(); i++) {
        if (i > 0) cout << " ";
        cout << positions[i];
    }
    cout << endl;
    return 0;
}
