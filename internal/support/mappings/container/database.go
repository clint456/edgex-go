/*******************************************************************************
* @microservice: support-notifications
* @author: Luo Qinwen, HY
* @version: 0.0.1
*******************************************************************************/
package container

import (
	"fmt"

	pgInfrastructure "github.com/clint456/edgex-go/internal/pkg/infrastructure/postgres"
	"github.com/clint456/edgex-go/internal/pkg/interfaces"
	mappingInterfaces "github.com/clint456/edgex-go/internal/support/mappings/infrastructure/interfaces"
	mappingPostgres "github.com/clint456/edgex-go/internal/support/mappings/infrastructure/postgres"
	"github.com/edgexfoundry/go-mod-bootstrap/v4/di"
)

// DBClientInterfaceName identifies the shared database client in the DIC.
var DBClientInterfaceName = di.TypeInstanceToName((*interfaces.DBClient)(nil))

// DBClientFrom returns the shared database client.
func DBClientFrom(get di.Get) interfaces.DBClient {
	return get(DBClientInterfaceName).(interfaces.DBClient)
}

func RepositoryFrom(get di.Get) (mappingInterfaces.Repository, error) {
	client, ok := DBClientFrom(get).(*pgInfrastructure.Client)
	if !ok {
		return nil, fmt.Errorf("support-mappings requires a PostgreSQL database client")
	}
	return mappingPostgres.NewRepository(client.ConnPool), nil
}
