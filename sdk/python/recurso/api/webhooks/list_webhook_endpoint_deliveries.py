from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.list_webhook_endpoint_deliveries_response_200 import ListWebhookEndpointDeliveriesResponse200
from ...models.list_webhook_endpoint_deliveries_status import ListWebhookEndpointDeliveriesStatus
from ...types import UNSET, Response, Unset


def _get_kwargs(
    id: UUID,
    *,
    limit: int | Unset = UNSET,
    offset: int | Unset = 0,
    status: ListWebhookEndpointDeliveriesStatus | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["limit"] = limit

    params["offset"] = offset

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/webhooks/{id}/deliveries".format(
            id=quote(str(id), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | ListWebhookEndpointDeliveriesResponse200 | None:
    if response.status_code == 200:
        response_200 = ListWebhookEndpointDeliveriesResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | ListWebhookEndpointDeliveriesResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    offset: int | Unset = 0,
    status: ListWebhookEndpointDeliveriesStatus | Unset = UNSET,
) -> Response[Error | ListWebhookEndpointDeliveriesResponse200]:
    """List deliveries for a webhook endpoint

     Recent delivery attempts to this endpoint, newest first.

    Args:
        id (UUID):
        limit (int | Unset):
        offset (int | Unset):  Default: 0.
        status (ListWebhookEndpointDeliveriesStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ListWebhookEndpointDeliveriesResponse200]
    """

    kwargs = _get_kwargs(
        id=id,
        limit=limit,
        offset=offset,
        status=status,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    offset: int | Unset = 0,
    status: ListWebhookEndpointDeliveriesStatus | Unset = UNSET,
) -> Error | ListWebhookEndpointDeliveriesResponse200 | None:
    """List deliveries for a webhook endpoint

     Recent delivery attempts to this endpoint, newest first.

    Args:
        id (UUID):
        limit (int | Unset):
        offset (int | Unset):  Default: 0.
        status (ListWebhookEndpointDeliveriesStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ListWebhookEndpointDeliveriesResponse200
    """

    return sync_detailed(
        id=id,
        client=client,
        limit=limit,
        offset=offset,
        status=status,
    ).parsed


async def asyncio_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    offset: int | Unset = 0,
    status: ListWebhookEndpointDeliveriesStatus | Unset = UNSET,
) -> Response[Error | ListWebhookEndpointDeliveriesResponse200]:
    """List deliveries for a webhook endpoint

     Recent delivery attempts to this endpoint, newest first.

    Args:
        id (UUID):
        limit (int | Unset):
        offset (int | Unset):  Default: 0.
        status (ListWebhookEndpointDeliveriesStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ListWebhookEndpointDeliveriesResponse200]
    """

    kwargs = _get_kwargs(
        id=id,
        limit=limit,
        offset=offset,
        status=status,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    offset: int | Unset = 0,
    status: ListWebhookEndpointDeliveriesStatus | Unset = UNSET,
) -> Error | ListWebhookEndpointDeliveriesResponse200 | None:
    """List deliveries for a webhook endpoint

     Recent delivery attempts to this endpoint, newest first.

    Args:
        id (UUID):
        limit (int | Unset):
        offset (int | Unset):  Default: 0.
        status (ListWebhookEndpointDeliveriesStatus | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ListWebhookEndpointDeliveriesResponse200
    """

    return (
        await asyncio_detailed(
            id=id,
            client=client,
            limit=limit,
            offset=offset,
            status=status,
        )
    ).parsed
