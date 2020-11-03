#
#     http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#
#   This source code is part of the near-RT RIC (RAN Intelligent Controller)
#   platform project (RICP).
#


set -e
set -x

# setup version tag
if [ -f ../container-tag.yaml ]
then
    tag=$(grep "tag:" ../container-tag.yaml | awk '{print $2}')
else
    tag="-"
fi

hash=$(git rev-parse --short HEAD || true)

export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH
export CFG_FILE=config/config-file.json
export RMR_SEED_RT=config/uta_rtg_ut.rt
GO111MODULE=on GO_ENABLED=0 GOOS=linux

# Build o1mediator
go build -a -installsuffix cgo -ldflags "-X main.Version=$tag -X main.Hash=$hash" -o o1agent ./cmd/o1agent.go

# Run o1agent UT
go test -v -p 1 -cover -coverprofile=/go/src/ws/agent/coverage.out ./...
