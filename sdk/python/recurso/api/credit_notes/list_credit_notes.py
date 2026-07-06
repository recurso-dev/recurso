from http import HTTPStatus
from typing import Any
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.list_credit_notes_response_200 import ListCreditNotesResponse200
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    customer_id: UUID | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    json_customer_id: str | Unset = UNSET
    if not isinstance(customer_id, Unset):
        json_customer_id = str(customer_id)
    params["customer_id"] = json_customer_id

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/credit-notes",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | ListCreditNotesResponse200 | None:
    if response.status_code == 200:
        response_200 = ListCreditNotesResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | ListCreditNotesResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    customer_id: UUID | Unset = UNSET,
) -> Response[Error | ListCreditNotesResponse200]:
    """List credit notes

    Args:
        customer_id (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ListCreditNotesResponse200]
    """

    kwargs = _get_kwargs(
        customer_id=customer_id,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    customer_id: UUID | Unset = UNSET,
) -> Error | ListCreditNotesResponse200 | None:
    """List credit notes

    Args:
        customer_id (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ListCreditNotesResponse200
    """

    return sync_detailed(
        client=client,
        customer_id=customer_id,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    customer_id: UUID | Unset = UNSET,
) -> Response[Error | ListCreditNotesResponse200]:
    """List credit notes

    Args:
        customer_id (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | ListCreditNotesResponse200]
    """

    kwargs = _get_kwargs(
        customer_id=customer_id,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    customer_id: UUID | Unset = UNSET,
) -> Error | ListCreditNotesResponse200 | None:
    """List credit notes

    Args:
        customer_id (UUID | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | ListCreditNotesResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            customer_id=customer_id,
        )
    ).parsed
