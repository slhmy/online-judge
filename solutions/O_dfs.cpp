#include <iostream>
#include <vector>
#include <algorithm>
using namespace std;

vector<vector<int>> adj;
vector<bool> visited;
vector<int> order;

void dfs(int u) {
    visited[u] = true;
    order.push_back(u);
    sort(adj[u].begin(), adj[u].end());
    for (int v : adj[u]) {
        if (!visited[v]) dfs(v);
    }
}

int main() {
    int n, m;
    cin >> n >> m;
    adj.resize(n + 1);
    visited.resize(n + 1, false);
    for (int i = 0; i < m; i++) {
        int u, v;
        cin >> u >> v;
        adj[u].push_back(v);
        adj[v].push_back(u);
    }
    dfs(1);
    for (int i = 0; i < (int)order.size(); i++) {
        if (i > 0) cout << " ";
        cout << order[i];
    }
    cout << endl;
    return 0;
}
