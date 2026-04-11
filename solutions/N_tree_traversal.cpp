// Given inorder and preorder traversal, output postorder traversal
#include <iostream>
#include <vector>
using namespace std;

vector<int> inord, preord, postord;
int preIdx = 0;

void build(int inL, int inR) {
    if (inL > inR) return;
    int root = preord[preIdx++];
    int mid = inL;
    while (inord[mid] != root) mid++;
    build(inL, mid - 1);
    build(mid + 1, inR);
    postord.push_back(root);
}

int main() {
    int n;
    cin >> n;
    inord.resize(n);
    preord.resize(n);
    for (int i = 0; i < n; i++) cin >> inord[i];
    for (int i = 0; i < n; i++) cin >> preord[i];
    build(0, n - 1);
    for (int i = 0; i < n; i++) {
        if (i > 0) cout << " ";
        cout << postord[i];
    }
    cout << endl;
    return 0;
}
